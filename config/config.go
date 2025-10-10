package config

import (
	"errors"
	"log"
	"os"

	yaml "gopkg.in/yaml.v2"
)

var _config *config

type NatsConfig struct {
	ConnectionString string `yaml:"connectionString" json:"-"`
	JWT              string `yaml:"jwt" json:"-"`
	Seed             string `yaml:"seed" json:"-"`
}

type RouterConfig struct {
	Port int    `yaml:"port" json:"port"`
	Mode string `yaml:"mode" json:"mode"`
}

type SystemConfig struct {
	Environment string `yaml:"environment" json:"environment"`
}

type ClusterConfig struct {
	Namespace         string `yaml:"namespace" json:"namespace"`
	CertManagerIssuer string `yaml:"certManagerIssuer" json:"certManagerIssuer"`
	ServiceName       string `yaml:"serviceName" json:"serviceName"`
	ServicePort       int32  `yaml:"servicePort" json:"servicePort"`
}

type config struct {
	NatsConfig    NatsConfig    `yaml:"nats" json:"nats"`
	RouterConfig  RouterConfig  `yaml:"router" json:"yaml"`
	SystemConfig  SystemConfig  `yaml:"system" json:"system"`
	ClusterConfig ClusterConfig `yaml:"cluster" json:"cluster"`
}

func (c *config) Nats() NatsConfig {
	return c.NatsConfig
}

func (c *config) Router() RouterConfig {
	return c.RouterConfig
}

func (c *config) System() SystemConfig {
	return c.SystemConfig
}

func (c *config) Cluster() ClusterConfig {
	return c.ClusterConfig
}

func (c *config) IsDevelopment() bool {
	return c.SystemConfig.Environment == "development"
}

func (c *config) validate() error {
	if c.NatsConfig.ConnectionString == "" {
		return errors.New("invalid NATS host, it can not be empty")
	}

	if c.SystemConfig.Environment == "" {
		return errors.New("invalid system environment, it can not be empty")
	}

	if c.ClusterConfig.Namespace == "" {
		return errors.New("cluster namespace cannot be empty")
	}

	if c.ClusterConfig.ServiceName == "" {
		return errors.New("cluster serviceName cannot be empty")
	}

	if c.ClusterConfig.ServicePort <= 0 {
		return errors.New("cluster servicePort must be a positive number")
	}

	// Note: CertManagerIssuer is optional and can be empty

	return nil
}

// Loads loads the Configuration file and verifies the settings
func Load(filePath string) error {
	c := config{}

	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
		return err
	}

	if err := yaml.Unmarshal(yamlFile, &c); err != nil {
		log.Fatalf("Unmarshal: %v", err)
		return err
	}

	_config = &c

	return c.validate()
}

func Config() *config {
	if _config == nil {
		log.Fatal("Configuration is not initialized, please call NewConfig first")
	}
	return _config
}
