/*
Copyright 2025 Farfetch

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"github.com/Farfetch/velero-plugin-artifactory/internal/plugin"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/framework"
)

func main() {
	framework.NewServer().
		RegisterObjectStore("farfetch/velero-plugin-artifactory", newObjectStorePlugin).
		Serve()
}

func newObjectStorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return plugin.NewObjectStore(logger), nil
}
