package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
		fmt.Printf("An error occured: %s\n", err)
	}
	err = yaml.Unmarshal(f, &config)
	if err != nil {
		fmt.Printf("An error occured while unmarshalling config: %s\n", err)
	}
	return config
}

func main() {
	config := getConfig()
	go func() {
		for {
			fmt.Println("Starting main loop iteration...")
			infoGauge.Reset()
			for i := range config.Contexts {
				resp, err := http.Get(config.SourceURL + config.Contexts[i])
				if err != nil {
					fmt.Printf("An error occured: %s\n", err)
				}
				var env environment
				decoder := json.NewDecoder(resp.Body)
				err = decoder.Decode(&env)
				if err != nil {
					fmt.Printf("An error occured: %s\n", err)
				}
				for a := range env.Applications {
					for c := range env.Applications[a].ContainersSpec {
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
			fmt.Println("Iteration done.")
			r, err := time.ParseDuration(config.RefreshInterval)
			if err != nil {
				fmt.Printf("An error occured: %s\n", err)
			}
			time.Sleep(r)
		}
	}()
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe("localhost:8080", nil)
	if err != nil {
		fmt.Printf("An error occured: %s\n", err)
		os.Exit(1)
	}
}
