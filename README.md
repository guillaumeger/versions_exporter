# versions_exporter

This app gets images tags from [sonar](https://stash.lapresse.ca/projects/DOCKER/repos/sonar/browse) and exposes them in prometheus format

## Configuration

versions_exporter takes a configuration file in yaml format with the following:

```
source_url: string <the url of sonar>
refresh_interval: time.Duration (https://golang.org/pkg/time/#Duration) <the interval to refresh the data from sonar>
contexts: slice <a list of sonar environments to get>
```

additionally it takes two environment variables, wich are mandatory:

```
VERSIONS_EXPORTER_LOGLEVEL= string <the loglevel of the application>
VERSIONS_EXPORTER_CONFIG_FILE= string <the full path to the configuration file>
```