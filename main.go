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

func getRefreshInterval() string {
	r, ok := os.LookupEnv("VERSIONS_EXPORTER_REFRESH_INTERVAL")
	if !ok {
		return "1h"
	}
	return r
}

func getAnnotationName() string {
	a, ok := os.LookupEnv("VERSIONS_EXPORTER_ANNOTATION_NAME")
	if !ok {
		return "versions_exporter/githubRepo"
	}
	return a
}

func getPort() string {
	p, ok := os.LookupEnv("VERSIONS_EXPORTER_PORT")
	if !ok {
		return "8083"
	}
	return p
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

func (ver versions) getDeploysVersions(c *kubernetes.Clientset) versions {
	log.Debugf("Getting current versions for deployments.")
	deploys, err := c.ExtensionsV1beta1().Deployments("").List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error getting deployments: %v.", err)
	}
	annotation := getAnnotationName()
	for d := range deploys.Items {
		v, ok := deploys.Items[d].Annotations[annotation]
		if ok {
			//if the label "app" is set we use this for application name, otherwise we use the name of the deploy
			n, ok := deploys.Items[d].Labels["app"]
			var appName string
			if ok {
				appName = n
			} else {
				appName = deploys.Items[d].Name
			}
			latestVersion := getLatestVersion(v)
			containers := deploys.Items[d].Spec.Template.Spec.Containers
			currentVersion := strings.Split(containers[0].Image, ":")[1]
			log.Debugf("Current version for application %v is %v", appName, currentVersion)
			ver = append(ver, versionMap{appName, currentVersion, latestVersion})
		}
	}
	return ver
}

func (ver versions) getDSVersions(c *kubernetes.Clientset) versions {
	log.Debugf("Getting current versions for daemonsets.")
	ds, err := c.AppsV1().DaemonSets("").List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error getting daemonsets: %v.", err)
	}
	annotation := getAnnotationName()
	for d := range ds.Items {
		v, ok := ds.Items[d].Annotations[annotation]
		if ok {
			//if the label "app" is set we use this for application name, otherwise we use the name of the ds
			n, ok := ds.Items[d].Labels["app"]
			var appName string
			if ok {
				appName = n
			} else {
				appName = ds.Items[d].Name
			}
			latestVersion := getLatestVersion(v)
			containers := ds.Items[d].Spec.Template.Spec.Containers
			currentVersion := strings.Split(containers[0].Image, ":")[1]
			log.Debugf("Current version for application %v is %v", appName, currentVersion)
			ver = append(ver, versionMap{appName, currentVersion, latestVersion})
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
			versions = versions.getDeploysVersions(clientset)
			versions = versions.getDSVersions(clientset)
			for _, v := range versions {
				log.Debugf("application name: %v, current version: %v, latest version: %v.", v.name, v.currentVersion, v.latestVersion)
				infoGauge.With(prometheus.Labels{
					"application_name": v.name,
					"current_version":  v.currentVersion,
					"latest_version":   v.latestVersion,
				}).Set(1)
			}
			log.Printf("Finished main loop")
			r, _ := time.ParseDuration(getRefreshInterval())
			log.Printf("Next iteration in %v", r)
			time.Sleep(r)
		}
	}()
	http.Handle("/metrics", promhttp.Handler())
	port := getPort()
	log.Infof("Serving /metrics on port %v", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("An error occured: %s", err)
	}
}
