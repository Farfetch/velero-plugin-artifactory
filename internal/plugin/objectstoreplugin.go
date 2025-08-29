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
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	articonfig "github.com/jfrog/jfrog-client-go/config"
	"github.com/sirupsen/logrus"
)

const defaultRoot = "/tmp/backups"

type ObjectStore struct {
	rtManager artifactory.ArtifactoryServicesManager
	labels    string
	log       logrus.FieldLogger
}

// NewObjectStore instantiates a ObjectStore.
func NewObjectStore(log logrus.FieldLogger) *ObjectStore {
	return &ObjectStore{log: log}
}

// Init prepares the ObjectStore for usage using the provided map of
// configuration key-value pairs. It returns an error if the ObjectStore
// cannot be initialized from the provided config.
func (f *ObjectStore) Init(config map[string]string) error {
	f.log.Infof("ObjectStore.Init called")

	// setup labels
	f.labels = config["labels"]

	// login to arti
	rtDetails := auth.NewArtifactoryDetails()
	rtDetails.SetUrl(config["url"])
	rtDetails.SetUser(config["user"])
	rtDetails.SetPassword(os.Getenv("ARTIFACTORY_PASSWORD"))
	rtDetails.SetApiKey(os.Getenv("ARTIFACTORY_API_KEY"))
	rtDetails.SetAccessToken(os.Getenv("ARTIFACTORY_ACCESS_TOKEN"))
	rtDetails.SetSshKeyPath(os.Getenv("ARTIFACTORY_SSH_KEY_PATH"))
	// if client certificates are required
	rtDetails.SetClientCertPath(os.Getenv("ARTIFACTORY_CLIENT_CERT_PATH"))
	rtDetails.SetClientCertKeyPath(os.Getenv("ARTIFACTORY_CLIENT_CERT_KEY_PATH"))

	// set default values
	dry_run := false
	if config["dry_run"] != "" {
		bool_run, err := strconv.ParseBool(config["dry_run"])
		if err != nil {
			return err
		}
		dry_run = bool_run
	}
	threads := 3
	if config["threads"] != "" {
		aux_threads, err := strconv.Atoi(config["threads"])
		if err != nil {
			return err
		}
		threads = aux_threads
	}
	dial_timeout := 30 * time.Second
	if config["dial_timeout"] != "" {
		timeout, err := strconv.Atoi(config["dial_timeout"])
		if err != nil {
			return err
		}
		dial_timeout = time.Duration(timeout) * time.Second
	}
	request_timeout := 10 * time.Minute
	if config["request_timeout"] != "" {
		timeout, err := strconv.Atoi(config["request_timeout"])
		if err != nil {
			return err
		}
		request_timeout = time.Duration(timeout) * time.Second
	}
	retries := 3
	if config["retries"] != "" {
		aux_retries, err := strconv.Atoi(config["retries"])
		if err != nil {
			return err
		}
		retries = aux_retries
	}

	serviceConfig, err := articonfig.NewConfigBuilder().
		SetServiceDetails(rtDetails).
		SetDryRun(dry_run).
		SetCertificatesPath(os.Getenv("ARTIFACTORY_CLIENT_CERT_PATH")).
		SetThreads(threads).
		SetDialTimeout(dial_timeout).
		SetOverallRequestTimeout(request_timeout).
		SetHttpRetries(retries).
		Build()
	if err != nil {
		return err
	}

	f.rtManager, err = artifactory.New(serviceConfig)
	if err != nil {
		return err
	}

	return nil
}

// PutObject creates a new object using the data in body within the specified
// object storage bucket with the given key.
func (f *ObjectStore) PutObject(bucket string, key string, body io.Reader) error {
	// Create file to upload
	path := filepath.Join(defaultRoot, bucket, key)

	log := f.log.WithFields(logrus.Fields{
		"repository": bucket,
		"key":        key,
		"path":       path,
	})
	log.Infof("PutObject")

	dir := filepath.Dir(path)
	log.Debugf("Creating dir %s", dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	log.Debug("Creating file")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	log.Debug("Writing to file")
	_, err = io.Copy(file, body)
	if err != nil {
		return err
	}

	// Upload
	params := services.NewUploadParams()
	// Specifies the local file system path to artifacts which should be uploaded to Artifactory.
	// You can specify multiple artifacts by using wildcards or a regular expression as designated by the Regexp param.
	params.Pattern = path
	// Specifies the target path in Artifactory in the following format: <repository name>/<repository path>
	// If the target path ends with a slash, the path is assumed to be a folder. For example, if you specify the target
	// as "repo-name/a/b/", then "b" is assumed to be a folder in Artifactory into which files should be uploaded.
	// If there is no terminal slash, the target path is assumed to be a file to which the uploaded file should be
	// renamed. For example, if you specify the target as "repo-name/a/b", the uploaded file is renamed to "b"
	// in Artifactory.
	params.Target = fmt.Sprintf("%s/%s", bucket, key)
	params.AddVcsProps = false
	// [Optional] Providing this option will collect and record build info for this build name and number.
	//params.BuildProps = "build.name=buildName;build.number=17;build.timestamp=1600856623553"
	// Set to false if you do not wish to collect artifacts in sub-folders to be uploaded to Artifactory.
	params.Recursive = true
	// Set to true to use a regular expression instead of wildcards expression to collect files to upload.
	params.Regexp = false
	// Set to true if you'd like to also apply the source path pattern for directories and not just for files.
	params.IncludeDirs = false
	// If set to false, files are uploaded according to their file system hierarchy.
	params.Flat = true
	// Set to true to extract an archive after it is deployed to Artifactory.
	params.ExplodeArchive = false

	if f.labels != "" {
		targetProps := utils.NewProperties()
		labels := strings.SplitSeq(f.labels, ";")
		for label := range labels {
			label_kv := strings.Split(label, "=")
			targetProps.AddProperty(label_kv[0], label_kv[1])
		}
		params.TargetProps = targetProps
	}

	uploadServiceOptions := artifactory.UploadServiceOptions{
		// Set to true to fail the upload operation if any of the files fail to upload
		FailFast: false,
	}

	log.Infof("Uploading files to Artifactory")
	totalUploaded, totalFailed, err := f.rtManager.UploadFiles(uploadServiceOptions, params)
	if err != nil {
		return err
	}
	log.Debugf("Files uploaded: %d", totalUploaded)
	log.Debugf("Failed uploads: %d", totalFailed)

	return nil
}

// ObjectExists checks if there is an object with the given key in the object storage bucket.
func (f *ObjectStore) ObjectExists(bucket, key string) (bool, error) {
	log := f.log.WithFields(logrus.Fields{
		"bucket": bucket,
		"key":    key,
	})
	log.Infof("ObjectExists")

	// check arti for existing object
	params := services.NewSearchParams()
	params.Pattern = fmt.Sprintf("%s/%s", bucket, key)
	// Filter the files by properties.
	params.Props = f.labels
	params.Recursive = true

	reader, err := f.rtManager.SearchFiles(params)
	if err != nil {
		return false, err
	}
	length, err := reader.Length()
	if err != nil {
		log.Error("Error getting search length.")
		return false, err
	}
	if length == 0 {
		return false, nil
	}
	defer reader.Close()
	return true, nil
}

// GetObject retrieves the object with the given key from the specified
// bucket in object storage.
func (f *ObjectStore) GetObject(bucket, key string) (io.ReadCloser, error) {
	log := f.log.WithFields(logrus.Fields{
		"bucket": bucket,
		"key":    key,
	})
	log.Infof("GetObject")
	path := filepath.Join(defaultRoot, bucket, key)

	// download from arti
	params := services.NewDownloadParams()
	// Specifies the source path in Artifactory, from which the artifacts should be downloaded,
	// in the format: <repository name>/<repository path>. Wildcards can be used to specify multiple artifacts.
	params.Pattern = fmt.Sprintf("%s/%s", bucket, strings.Trim(key, "/"))
	// Optional argument specifying the local file system target path.
	// If the target path ends with a slash, it is assumed to be a directory.
	// If there is no terminal slash, the target path is assumed to be a file.
	params.Target = path
	// Filter the downloaded files by properties.
	params.Props = f.labels
	params.Flat = true
	//params.Recursive = true
	//params.IncludeDirs = false
	//params.Explode = false
	//params.Symlink = true
	//params.ValidateSymlink = false
	totalDownloaded, totalFailed, err := f.rtManager.DownloadFiles(params)
	if err != nil {
		return nil, err
	}
	log.Debugf("Files downloaded: %d", totalDownloaded)
	log.Debugf("Failed downloads: %d", totalFailed)

	// verify this
	return os.Open(path)
}

// ListCommonPrefixes gets a list of all object key prefixes that start with
// the specified prefix and stop at the next instance of the provided delimiter.
//
// For example, if the bucket contains the following keys:
//
//	a-prefix/foo-1/bar
//	a-prefix/foo-1/baz
//	a-prefix/foo-2/baz
//	some-other-prefix/foo-3/bar
//
// and the provided prefix arg is "a-prefix/", and the delimiter is "/",
// this will return the slice {"a-prefix/foo-1/", "a-prefix/foo-2/"}.
func (f *ObjectStore) ListCommonPrefixes(bucket, prefix, delimiter string) ([]string, error) {
	log := f.log.WithFields(logrus.Fields{
		"bucket":    bucket,
		"delimiter": delimiter,
		"prefix":    prefix,
	})
	log.Infof("ListCommonPrefixes")

	params := services.NewSearchParams()
	// Specifies the search path in Artifactory, in the following format: <repository name>/<repository path>.
	// You can use wildcards to specify multiple artifacts.
	params.Pattern = fmt.Sprintf("%s/%s*", bucket, prefix)
	// Filter the files by properties.
	params.Props = f.labels
	params.Recursive = true

	reader, err := f.rtManager.SearchFiles(params)
	if err != nil {
		return nil, err
	}

	// Iterate over the results.
	var objectsList []string
	for currentResult := new(utils.ResultItem); reader.NextRecord(currentResult) == nil; currentResult = new(utils.ResultItem) {
		artifact := fmt.Sprintf("%s/%s/%s", currentResult.Repo, currentResult.Path, currentResult.Name)
		beginningS := fmt.Sprintf("%s/%s", bucket, prefix)

		// remove bucket + prefix from result path
		subKey := artifact[len(beginningS):]
		delimited := strings.Split(subKey, delimiter)

		// object = prefix + first split until delimiter + delimiter
		object := fmt.Sprintf("%s%s%s", prefix, delimited[0], delimiter)
		// append only if not exists
		if !slices.Contains(objectsList, object) {
			objectsList = append(objectsList, object)
		}
	}

	defer reader.Close()
	return objectsList, nil
}

// ListObjects gets a list of all keys in the specified bucket
// that have the given prefix.
func (f *ObjectStore) ListObjects(bucket, prefix string) ([]string, error) {
	log := f.log.WithFields(logrus.Fields{
		"bucket": bucket,
		"prefix": prefix,
	})
	log.Infof("ListObjects")

	objects, err := f.ListCommonPrefixes(bucket, prefix, "/")
	if err != nil {
		return nil, err
	}
	return objects, nil
}

// DeleteObject removes the object with the specified key from the given
// bucket.
func (f *ObjectStore) DeleteObject(bucket, key string) error {
	log := f.log.WithFields(logrus.Fields{
		"bucket": bucket,
		"key":    key,
	})
	log.Infof("DeleteObject")
	params := services.NewDeleteParams()
	params.Pattern = fmt.Sprintf("%s/%s", bucket, strings.Trim(key, "/"))
	// Filter the files by properties.
	params.Props = f.labels
	params.Recursive = true

	pathsToDelete, err := f.rtManager.GetPathsToDelete(params)
	if err != nil {
		return err
	}
	defer pathsToDelete.Close()
	totalDeleted, err := f.rtManager.DeleteFiles(pathsToDelete)
	if err != nil {
		log.Error("Failed deleteing files")
		return err
	}
	log.Debugf("Files deleted: %d", totalDeleted)

	return nil
}

// CreateSignedURL creates a pre-signed URL for the given bucket and key that expires after ttl.
func (f *ObjectStore) CreateSignedURL(bucket, key string, ttl time.Duration) (string, error) {
	log := f.log.WithFields(logrus.Fields{
		"bucket": bucket,
		"key":    key,
	})
	log.Infof("CreateSignedURL")

	jconfig := f.rtManager.GetConfig().GetServiceDetails()

	cred := ""
	if jconfig.GetPassword() != "" {
		cred = jconfig.GetPassword()
	}
	if jconfig.GetApiKey() != "" {
		cred = jconfig.GetApiKey()
	}
	if jconfig.GetAccessToken() != "" {
		cred = jconfig.GetAccessToken()
	}
	u, err := url.Parse(jconfig.GetUrl())
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s:%s@%s%s%s/%s", url.QueryEscape(jconfig.GetUser()), url.QueryEscape(cred), u.Host, u.Path, bucket, key), nil
}
