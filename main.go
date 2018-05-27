package main

import (
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/tennix/k8swatch/pkg/controller"
	"github.com/tennix/k8swatch/pkg/handlers"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig      string
	master          string
	alertmanagerURL string
	configFile      string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file, omit this if in cluster")
	flag.StringVar(&configFile, "config-file", "", "path to filter rules file")
	flag.StringVar(&master, "master", "http://127.0.0.1:8080", "master url")
	flag.StringVar(&alertmanagerURL, "alertmanager", "http://127.0.0.1:9093", "alertmanager url")
	flag.Parse()
}

func main() {
	var cfg *rest.Config
	var err error
	if kubeconfig == "" {
		cfg, err = rest.InClusterConfig()
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		glog.Fatalf("failed to load kubeconfig: %v", err)
	}
	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to initialize kube client: %v", err)
	}

	var config handlers.Config
	if configFile == "" {
		glog.Infof("empty config file, all event will fire an alert")
	} else {
		file, err := os.Open(configFile)
		if err != nil {
			glog.Fatalf("failed to open config file: %v", err)
		}
		defer file.Close()

		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(&config); err != nil {
			glog.Fatalf("failed to unmarshal config file: %v", err)
		}
	}

	alertmanager := handlers.New(alertmanagerURL, config)
	controller.Start(kubeCli, alertmanager)
}
