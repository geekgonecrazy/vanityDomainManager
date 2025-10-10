package main

import (
	"os"
	"testing"

	"github.com/geekgonecrazy/vanityDomainManager/config"
)

func TestClusterConfigLoading(t *testing.T) {
	// Test loading a valid config
	err := config.Load("../../config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	clusterConfig := config.Config().Cluster()

	// Verify the values are loaded correctly
	if clusterConfig.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", clusterConfig.Namespace)
	}

	if clusterConfig.CertManagerIssuer != "letsencrypt-prod" {
		t.Errorf("Expected certManagerIssuer 'letsencrypt-prod', got '%s'", clusterConfig.CertManagerIssuer)
	}

	if clusterConfig.ServiceName != "your-service" {
		t.Errorf("Expected serviceName 'your-service', got '%s'", clusterConfig.ServiceName)
	}

	if clusterConfig.ServicePort != 3000 {
		t.Errorf("Expected servicePort 3000, got %d", clusterConfig.ServicePort)
	}
}

func TestClusterConfigValidation(t *testing.T) {
	// Create a temporary config file with invalid cluster config
	tempConfigPath := "/tmp/test-invalid-cluster.yaml"
	
	// This should fail validation
	invalidConfig := `
system:
  environment: "development"
router:
  port: 3887
  mode: "debug"
nats:
  connectionString: "nats://localhost:4222"
  jwt: "asdf"
  seed: "asdf"
cluster:
  namespace: ""
  certManagerIssuer: ""
  serviceName: ""
  servicePort: 0
`
	
	// Write the invalid config to a temp file
	if err := writeToFile(tempConfigPath, invalidConfig); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	
	// Try to load it - should fail
	err := config.Load(tempConfigPath)
	if err == nil {
		t.Error("Expected config validation to fail, but it passed")
	}
}

func writeToFile(path, content string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	_, err = file.WriteString(content)
	return err
}