package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/onrik/logrus/filename"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var infoGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "application_info",
	Help: "Informations on applications, especially version.",
}, []string{
	"application_name",
	"current_version",
	"latest_version",
})

type versionMap struct {
	name           string
	currentVersion string
	latestVersion  string
}

type versions []versionMap

func init() {
	logLevels := map[string]log.Level{
		"panic": log.PanicLevel,
		"fatal": log.FatalLevel,
		"error": log.ErrorLevel,
		"warn":  log.WarnLevel,
		"info":  log.InfoLevel,
		"debug": log.DebugLevel,
	}
	logLevel, ok := os.LookupEnv("VERSIONS_EXPORTER_LOGLEVEL")
	if !ok {
		log.SetLevel(logLevels["error"])
	} else {
		log.SetLevel(logLevels[logLevel])
	}
	log.AddHook(filename.NewHook())
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	log.SetOutput(os.Stdout)
}

func getDefaultValue(envVar, def string) string {
	r, ok := os.LookupEnv(envVar)
	if !ok {
		return def
	}
	return r
}

func getLatestVersion(repo string) string {
	log.Debugf("Getting latest version of repo %v from github.", repo)
	sepRepo := strings.Split(repo, "/")
	client := github.NewClient(nil)
	version, _, err := client.Repositories.GetLatestRelease(context.Background(), sepRepo[0], sepRepo[1])
	if err != nil {
		log.Errorf("An error occured: %v.", err)
		return ""
	}
	return *version.TagName
}

func (ver versions) getPodsVersions(c *kubernetes.Clientset) versions {
	log.Debugf("Getting current versions for pods.")
	pods, err := c.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error getting pods: %v.", err)
	}
	annotation := getDefaultValue("VERSIONS_EXPORTER_ANNOTATION_NAME", "versions-exporter/githubRepo")
	log.Debugf("Using annotation %v", annotation)
	for p := range pods.Items {
		v, ok := pods.Items[p].Annotations[annotation]
		if ok {
			appName := pods.Items[p].Spec.Containers[0].Name
			latestVersion := getLatestVersion(v)
			containers := pods.Items[p].Spec.Containers
			currentVersion := strings.Split(containers[0].Image, ":")[1]
			log.Debugf("Current version for application %v is %v", appName, currentVersion)
			ver = append(ver, versionMap{appName, currentVersion, latestVersion})
		}
	}
	return ver
}

func (ver versions) getCustomContainersVersions(c *kubernetes.Clientset) versions {
	log.Debugf("Getting custom containers versions.")
	pods, err := c.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error getting pods: %v.", err)
	}
	for p := range pods.Items {
		for k, v := range pods.Items[p].Annotations {
			if strings.Split(k, "/")[0] == "versions-exporter" && strings.Split(k, "/")[1] != "githubRepo" {
				appName := strings.Split(k, "/")[1]
				latestVersion := getLatestVersion(v)
				var currentVersion string
				for container := range pods.Items[p].Spec.Containers {
					if pods.Items[p].Spec.Containers[container].Name == appName {
						currentVersion = strings.Split(pods.Items[p].Spec.Containers[container].Image, ":")[1]
					}
				}
				ver = append(ver, versionMap{appName, currentVersion, latestVersion})
			}
		}
	}
	return ver
}

func createK8sClient() *kubernetes.Clientset {
	var conf *restclient.Config
	var err error
	c, ok := os.LookupEnv("VERSIONS_EXPORTER_OUT_OF_CLUSTER")
	if ok {
		if c == "true" {
			conf, err = clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
			if err != nil {
				log.Fatalf("Could not create k8s client: %v", err)
			}
		} else {
			conf, err = rest.InClusterConfig()
			if err != nil {
				log.Fatalf("Could not create k8s client: %v", err)
			}
		}
	} else {
		conf, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Could not create k8s client: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		panic(err.Error)
	}
	return clientset
}

func main() {
	go func() {
		for {
			log.Printf("Starting main loop iteration")
			infoGauge.Reset()
			var versions versions
			clientset := createK8sClient()
			versions = versions.getPodsVersions(clientset)
			versions = versions.getCustomContainersVersions(clientset)
			for _, v := range versions {
				log.Debugf("application name: %v, current version: %v, latest version: %v.", v.name, v.currentVersion, v.latestVersion)
				infoGauge.With(prometheus.Labels{
					"application_name": v.name,
					"current_version":  v.currentVersion,
					"latest_version":   v.latestVersion,
				}).Set(1)
			}
			log.Printf("Finished main loop")
			r, _ := time.ParseDuration(getDefaultValue("VERSIONS_EXPORTER_REFRESH_INTERVAL", "1h"))
			log.Printf("Next iteration in %v", r)
			time.Sleep(r)
		}
	}()
	http.Handle("/metrics", promhttp.Handler())
	port := getDefaultValue("VERSIONS_EXPORTER_METRICS_PORT", "8083")
	log.Infof("Serving /metrics on port %v", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("An error occured: %s", err)
	}
}
