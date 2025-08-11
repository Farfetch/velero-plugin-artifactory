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

package plugin

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

var testConfig = map[string]string{
	"url":          os.Getenv("ARTIFACTORY_URL"),
	"user":         os.Getenv("ARTIFACTORY_USERNAME"),
	"pass":         os.Getenv("ARTIFACTORY_PASSWORD"),
	"bucket":       os.Getenv("BUCKET"),
	"key":          os.Getenv("KEY"),
	"testFilePath": os.Getenv("TEST_FILE_PATH"),
}

func artifactoryLogin(t *testing.T, objectStore *ObjectStore) {
	err := objectStore.Init(testConfig)
	if err != nil {
		t.Error(err)
	}
}

func TestInitPassword(t *testing.T) {
	objectStore := NewObjectStore(logrus.New())

	err := objectStore.Init(testConfig)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPutObject(t *testing.T) {
	objectStore := NewObjectStore(logrus.New())

	artifactoryLogin(t, objectStore)

	dir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	path := filepath.Join(dir, testConfig["testFilePath"])
	fd, err := os.Open(path)
	if err != nil {
		t.Error(err)
	}
	defer fd.Close()

	err = objectStore.PutObject(testConfig["bucket"], testConfig["key"], fd)
	if err != nil {
		t.Error(err)
	}
}

func TestListObjects(t *testing.T) {
	objectStore := NewObjectStore(logrus.New())

	artifactoryLogin(t, objectStore)

	expected := []string{"backups/"}
	objects, err := objectStore.ListObjects(testConfig["bucket"], "")
	if err != nil {
		t.Error(err)
	}
	if len(objects) == 0 {
		t.Error("No objects found")
	}
	if !slices.Equal(expected, objects) {
		t.Error("Objects found not expected")
	}

	keySplit := strings.Split(testConfig["key"], "/")
	expected = []string{fmt.Sprintf("backups/%s/", keySplit[1])}
	objects, err = objectStore.ListObjects(testConfig["bucket"], "backups/")
	if err != nil {
		t.Error(err)
	}
	if len(objects) == 0 {
		t.Error("No objects found")
	}
	if !slices.Equal(expected, objects) {
		t.Error("Objects found not expected")
	}
}

func TestObjectExists(t *testing.T) {
	objectStore := NewObjectStore(logrus.New())

	artifactoryLogin(t, objectStore)

	exists, err := objectStore.ObjectExists(testConfig["bucket"], testConfig["key"])
	if err != nil {
		t.Error(err)
	}

	if !exists {
		t.Fail()
	}
}

func TestGetObject(t *testing.T) {
	objectStore := NewObjectStore(logrus.New())

	artifactoryLogin(t, objectStore)

	rc, err := objectStore.GetObject(testConfig["bucket"], testConfig["key"])
	if err != nil {
		t.Error(err)
	}

	path := filepath.Join(defaultRoot, testConfig["bucket"], testConfig["key"])
	outputPath := path + "-output"
	fd, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		t.Error(err)
	}

	bw, err := io.Copy(fd, rc)
	if err != nil {
		t.Error(err)
	}

	os.Remove(path) // failing to delete file should not fail the test
	t.Logf("bytes written: %d", bw)
}

func TestDeleteObject(t *testing.T) {
	objectStore := NewObjectStore(logrus.New())

	artifactoryLogin(t, objectStore)

	err := objectStore.DeleteObject(testConfig["bucket"], testConfig["key"])
	if err != nil {
		t.Error(err)
	}
}
