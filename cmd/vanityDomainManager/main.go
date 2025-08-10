package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/geekgonecrazy/vanityDomainManager/config"
	"github.com/geekgonecrazy/vanityDomainManager/kubernetes"
	"github.com/geekgonecrazy/vanityDomainManager/queueManager"
	"github.com/geekgonecrazy/vanityDomainManager/router"
)

func main() {
	var kubeconfig *string
	if home := os.Getenv("HOME"); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	configPath := flag.String("config", "config.yaml", "path to the configuration file")

	flag.Parse()

	if err := kubernetes.NewClient(*kubeconfig); err != nil {
		panic(err)
	}

	if err := config.Load(*configPath); err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	mgr, err := queueManager.Start()
	if err != nil {
		panic(fmt.Errorf("failed to setup NATS: %w", err))
	}

	if err := mgr.StartWorkers(); err != nil {
		panic(err)
	}

	router.Start()

}
