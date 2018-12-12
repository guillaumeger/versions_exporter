# versions_exporter

This app takes the current version of an app in k8s, gets the latest release from github, and expose them via prometheus format metrics.

It scans deployments and daemonsets for an annotation that specifies the github org/repo.

## Configuration

versions_exporter takes all of its configuration via env variables.

|variable name | type |usage | default value|
|--------------|------|------|--------------|
|VERSIONS_EXPORTER_LOGLEVEL | string | Specifies the log level. Possible values are: `panic`, `fatal`, `error`, `warn`, `info` and `debug`. | `error`|
|VERSIONS_EXPORTER_REFRESH_INTERVAL | [time.Duration](https://golang.org/pkg/time/#Duration) | The interval of time between each scan | `1h` |
|VERSIONS_EXPORTER_OUT_OF_CLUSTER | boolean | By default, versions_exporter is designed to run inside the k8s cluster, but it can also run outside by setting this var to `true`. It expects a valid kube config file. | `false`
|VERSIONS_EXPORTER_ANNOTATION_NAME | string | The annotation name that will specify the github repo | `versions_exporter/githubRepo`|
|VERSIONS_EXPORTER_PORT | string | the port that will be used to expose metrics | `8083`|

## Limitations

- In the case of a deployment or daemonset with multiple pods, versions_exporter will only take the first pod to get the current version. So you should always specify the main application first.
- For the moment only deployment and daemonset are supported
- It is strongly encouraged to add an `app` label to your deployment, as versions_exporter will use this label for the application name. Otherwise, it will use the name of the deployment, wich can lead to some unwanted names, with helm for instance.
- Github as a rate limit of 60/h on api calls for unauthenticated users, so plan accordingly!