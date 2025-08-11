# Velero Artifactory Plugins

JFrog Artifactory ObjectStore plugins for Velero.

This plugin focuses on ObjectStore backups and has no implementation for Volumes.

## Compatibility

This plugin was developed and tested with Artifactory version 7.111.X and Velero 1.16.X

| Plugin Version | Velero Version | Artifactory Version |
| :------------- | :------------- | :------------------ |
| v0.1.x         | 1.16.x         | 7.111.X             |

This section will be updated as further release are made.

## Artifactory Authentication

For authenticating with Artifactory we need the artifactory URL, a username and a secret to login with.

The username and artifactory URL are to be setup in the BackupStorageLocation object. The secret should be made available as an environment variable.

Authentication is made using the available Artifactory options, namely:
* password (env: ARTIFACTORY_PASSWORD)
* API key (env: ARTIFACTORY_API_KEY)
* access key (env: ARTIFACTORY_ACCESS_TOKEN)
* SSH key (env: ARTIFACTORY_SSH_KEY_PATH)

## Setup

You can use any conventional way to install the plugin, either using the CLI or an Helm chart.

The first piece to have is the BackupStorageLocation well configured, as seen in this example:

```yaml
apiVersion: velero.io/v1
kind: BackupStorageLocation
metadata:
  name: backup1
  namespace: velero
spec:
  config:
    url: https://example.com/artifactory
    user: backup-user
    # optional
    labels: label1=label;label2=another-label;label3=third label
  default: true
  objectStorage:
    bucket: kontractor-backups
  provider: farfetch/velero-plugin-artifactory
```

The last piece needed is the credential to be passed as an environment value from a secret in the velero container. Example:

```yaml
(...)
- name: ARTIFACTORY_ACCESS_TOKEN
  valueFrom:
    secretKeyRef:
      key: access_key
      name: artifactory-auth
(...)
```

### Labels

Artifactory has the option to provide properties to it's artifacts and so this plugin offers the same option. You can add any number of labels with the `labels` option in the config.

Labels should be a semi-colon seperated list of key-value pairs, with virtually any character in them (see example above). Since the string parsing is made using `;` and `=` as delimiters these need to be avoided.

## Building the plugin locally

To build the plugin, run

```bash
$ make build
```

To build the image, run

```bash
$ make container
```

You customize the docker build with:
* IMAGE - default is `farfetch/velero-plugin-artifactory`
* VERSION - default is `main`
* PLATFORM - default is `linux/amd64`

Example:

```bash
$ IMAGE=your-repo/your-name VERSION=your-version-tag PLATFORM=linux/arm64 make container 
```

## Tests

Testing is done against a locally deployed Artifactory instance. This was setup using docker and the Makefile has instruction to boot it up.

Before running the tests you need to create an `.env` file with the needed configuration. You can find a [template](.env_template) file with the basic configuration.

You can find the needed default credentials of the locally running artifactory in JFrog's official [documentation](https://jfrog.com/help/r/jfrog-installation-setup-documentation/docker-compose-next-steps).

To run the tests simply run:

```bash
make test
```

This will setup an Artifactory instance locally and run the integration tests that were written.

Afterwards you can run:

```bash
make clean
```

to cleanup your environment from any files that were outputted and to stop and remove the local artifactory instance.

## Contributing

Read our [contributing guidelines](CONTRIBUTING.md) to learn about our development process, how to propose bugfixes and improvements, and how to build and test your changes.

## Development

The plugin interface is built based on the [official Velero plugin example](https://github.com/vmware-tanzu/velero-plugin-example).
