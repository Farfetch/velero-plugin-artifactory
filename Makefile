# Copyright 2025 Farfetch
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
# 	 http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

include .env
export

REGISTRY ?= farfetch
IMAGE    ?= $(REGISTRY)/velero-plugin-artifactory
VERSION  ?= main
GIT_SHA  ?= somesha

PLATFORM ?= linux/amd64

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

ARTIFACTORY_VERSION ?= 7.111.9

UID ?= $(shell id -u)
GID ?= $(shell id -g)

# builds the binary using 'go build' in the local environment.
.PHONY: build
build: build-dirs
	@echo ""
	@echo "======================================================================"
	@echo " 👷  Building package"
	@echo "======================================================================"
	@echo ""
	CGO_ENABLED=0 go build -v -o _output/bin/$(GOOS)/$(GOARCH) .

.PHONY: setup-arti
setup-arti:
	@echo ""
	@echo "======================================================================"
	@echo " 🏭  Setting up Local Artifactory for testing"
	@echo "======================================================================"
	@echo ""
	mkdir -p ./tests/artifactory/var/etc/
	envsubst < ./tests/system.yaml > ./tests/artifactory/var/etc/system.yaml
	UID=${UID} GID=${GID} docker compose -f tests/docker-compose.yaml -p local_arti_tests up -d
	@echo ""
	@echo "======================================================================"
	@echo " ⏰  Waiting for Artifactory to be up"
	@echo "======================================================================"
	@echo ""
	timeout 300 bash -c 'while [[ "$$(curl -s -o /dev/null -w ''%{http_code}'' 127.0.0.1:8082/router/api/v1/system/health)" != "200" ]]; do sleep 5; done' || false

# test runs unit tests using 'go test' in the local environment.
.PHONY: test
test: setup-arti
	@echo ""
	@echo "======================================================================"
	@echo " 🐞  Running integration tests"
	@echo "======================================================================"
	@echo ""
	CGO_ENABLED=0 go test -v -timeout 60s ./...

# container builds a Docker image containing the binary.
.PHONY: container
container:
	@echo ""
	@echo "======================================================================"
	@echo " 🐳  Building container image"
	@echo "======================================================================"
	@echo ""
	docker buildx build \
		--platform $(PLATFORM) \
		--tag $(IMAGE):$(VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_SHA=$(GIT_SHA) \
		--no-cache \
		.

# push pushes the Docker image to its registry.
.PHONY: push
push:
	@echo ""
	@echo "======================================================================"
	@echo " 📤  Pushing image to registry"
	@echo "======================================================================"
	@echo ""
	@docker push $(IMAGE):$(VERSION)
ifeq ($(TAG_LATEST), true)
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest
	docker push $(IMAGE):latest
endif

.PHONY: cleanup-local-arti
cleanup-local-arti:
	@echo ""
	@echo "======================================================================"
	@echo " 🧹  Cleaning up Local Artifactory instance"
	@echo "======================================================================"
	@echo ""
	docker compose -p local_arti_tests down
	rm -rf ./tests/artifactory

# clean removes build artifacts from the local environment.
.PHONY: clean
clean: cleanup-local-arti
	@echo ""
	@echo "======================================================================"
	@echo " 🧹  Cleaning up output files"
	@echo "======================================================================"
	@echo ""
	rm -rf _output
	rm -rf /tmp/backups

# build-dirs creates the necessary directories for a build in the local environment.
.PHONY: build-dirs
build-dirs:
	@mkdir -p _output/bin/$(GOOS)/$(GOARCH)
