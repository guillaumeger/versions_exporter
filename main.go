package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/onrik/logrus/filename"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var infoGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "application_info",
	Help: "Informations on applications, especially version.",
}, []string{
	"application_name",
	"version",
	"environment",
	"repository",
	"container_name",
})

type versionMap struct {
	name           string
	currentVersion string
	latestVersion  string
}

type versions []versionMap

func logConfig() {
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
}

func getLatestVersion(repo string) string {
	sepRepo := strings.Split(repo, "/")
	client := github.NewClient(nil)
	version, _, err := client.Repositories.GetLatestRelease(context.Background(), sepRepo[0], sepRepo[1])
	if err != nil {
		panic(err.Error)
	}
	return *version.TagName
}

func (ver versions) getDeploysVersions(c *kubernetes.Clientset) versions {
	deploys, err := c.ExtensionsV1beta1().Deployments("").List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for d := range deploys.Items {
		v, ok := deploys.Items[d].Annotations["nuglif.net/upstreamProject"]
		if ok {
			latestVersion := getLatestVersion(v)
			containers := deploys.Items[d].Spec.Template.Spec.Containers
			currentVersion := strings.Split(containers[0].Image, ":")[1]
			ver = append(ver, versionMap{deploys.Items[d].Name, currentVersion, latestVersion})
		}
	}
	return ver
}

func (ver versions) getDSVersions(c *kubernetes.Clientset) versions {
	ds, err := c.AppsV1().Deployments("").List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for d := range ds.Items {
		v, ok := ds.Items[d].Annotations["nuglif.net/upstreamProject"]
		if ok {
			latestVersion := getLatestVersion(v)
			containers := ds.Items[d].Spec.Template.Spec.Containers
			currentVersion := strings.Split(containers[0].Image, ":")[1]
			ver = append(ver, versionMap{ds.Items[d].Name, currentVersion, latestVersion})
		}
	}
	return ver
}

func main() {
	logConfig()
	var versions versions
	conf, err := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		panic(err.Error)
	}
	versions = versions.getDeploysVersions(clientset)
	versions = versions.getDSVersions(clientset)
	fmt.Println(versions)
}

//	go func() {
//		for {
//			log.Printf("Starting main loop iteration...")
//			infoGauge.Reset()
//			for i := range config.Contexts {
//				log.Debugf("Getting env %s", config.Contexts[i])
//				resp, err := http.Get(config.SourceURL + config.Contexts[i])
//				if err != nil {
//					log.Errorf("An error occured: %s\n", err)
//				}
//				var env environment
//				decoder := json.NewDecoder(resp.Body)
//				err = decoder.Decode(&env)
//				if err != nil {
//					log.Errorf("An error occured: %s\n", err)
//				}
//				for a := range env.Applications {
//					log.Debugf("Getting application %s", env.Applications[a])
//					for c := range env.Applications[a].ContainersSpec {
//						log.Debugf("Getting container %s", env.Applications[a].ContainersSpec)
//						infoGauge.With(prometheus.Labels{
//							"application_name": env.Applications[a].Name,
//							"version":          env.Applications[a].ContainersSpec[c].Tag,
//							"environment":      config.Contexts[i],
//							"repository":       env.Applications[a].ContainersSpec[c].Repository,
//							"container_name":   env.Applications[a].ContainersSpec[c].Name,
//						}).Set(1)
//					}
//				}
//			}
//			log.Printf("Iteration done.")
//			log.Printf("Next iteration in %v.", config.RefreshInterval)
//			r, err := time.ParseDuration(config.RefreshInterval)
//			if err != nil {
//				log.Errorf("An error occured: %s\n", err)
//			}
//			time.Sleep(r)
//		}
//	}()
//	http.Handle("/metrics", promhttp.Handler())
//	err := http.ListenAndServe(":8083", nil)
//	if err != nil {
//		log.Fatalf("An error occured: %s\n", err)
//	}
