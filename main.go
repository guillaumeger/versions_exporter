package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/onrik/logrus/filename"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type container struct {
	Name       string `json:"name,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

type containerStatus struct {
	container
	Ready bool `json:"ready,omitempty"`
}

type pod struct {
	Name              string            `json:"name,omitempty"`
	ContainerStatuses []containerStatus `json:"containerStatuses"`
}

type app struct {
	Name                string      `json:"name,omitempty"`
	Chart               string      `json:"chart,omitempty"`
	Replicas            int         `json:"replicas,omitempty"`
	UnavailableReplicas int         `json:"unavailableReplicas,omitempty"`
	ContainersSpec      []container `json:"containersSpec,omitempty"`
	Pods                []pod       `json:"pods,omitempty"`
}

type environment struct {
	Environment  string `json:"environment,omitempty"`
	Namespace    string `json:"namespace,omitempty"`
	Applications []app  `json:"applications,omitempty"`
}

type config struct {
	SourceURL       string   `yaml:"source_url,omitempty"`
	RefreshInterval string   `yaml:"refresh_interval,omitempty"`
	Contexts        []string `yaml:"contexts,omitempty"`
}

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

func getConfig() config {
	var config config
	f, err := ioutil.ReadFile(os.Getenv("VERSIONS_EXPORTER_CONFIG_FILE"))
	if err != nil {
		log.Fatalf("An error occured while reading config file: %s\n", err)
	}
	err = yaml.Unmarshal(f, &config)
	if err != nil {
		log.Fatalf("An error occured while unmarshalling config: %s\n", err)
	}
	return config
}

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

func main() {
	logConfig()
	config := getConfig()
	go func() {
		for {
			log.Printf("Starting main loop iteration...")
			infoGauge.Reset()
			for i := range config.Contexts {
				log.Debugf("Getting env %s", config.Contexts[i])
				resp, err := http.Get(config.SourceURL + config.Contexts[i])
				if err != nil {
					log.Errorf("An error occured: %s\n", err)
				}
				var env environment
				decoder := json.NewDecoder(resp.Body)
				err = decoder.Decode(&env)
				if err != nil {
					log.Errorf("An error occured: %s\n", err)
				}
				for a := range env.Applications {
					log.Debugf("Getting application %s", env.Applications[a])
					for c := range env.Applications[a].ContainersSpec {
						log.Debugf("Getting container %s", env.Applications[a].ContainersSpec)
						infoGauge.With(prometheus.Labels{
							"application_name": env.Applications[a].Name,
							"version":          env.Applications[a].ContainersSpec[c].Tag,
							"environment":      config.Contexts[i],
							"repository":       env.Applications[a].ContainersSpec[c].Repository,
							"container_name":   env.Applications[a].ContainersSpec[c].Name,
						}).Set(1)
					}
				}
			}
			log.Printf("Iteration done.")
			log.Printf("Next iteration in %v.", config.RefreshInterval)
			r, err := time.ParseDuration(config.RefreshInterval)
			if err != nil {
				log.Errorf("An error occured: %s\n", err)
			}
			time.Sleep(r)
		}
	}()
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("An error occured: %s\n", err)
	}
}
