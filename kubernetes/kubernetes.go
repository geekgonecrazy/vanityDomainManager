package kubernetes

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/geekgonecrazy/vanityDomainManager/config"
	"github.com/geekgonecrazy/vanityDomainManager/jobs"
	v1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeClient *KubeClient

type KubeClient struct {
	client            *kubernetes.Clientset
	Namespace         string
	CertManagerIssuer string
	ServiceName       string
	ServicePort       int32
}

func GetClient() *KubeClient {
	return kubeClient
}

func NewClient(kubeconfig string, clusterConfig config.ClusterConfig) (err error) {
	var config *rest.Config

	if kubeconfig != "" {
		// Use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("Failed to create kubeconfig: %v", err)
		}
	} else {
		// Creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("Failed to create in-cluster config: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Failed to create clientset: %v", err)
	}

	kubeClient = &KubeClient{
		client:            clientset,
		Namespace:         clusterConfig.Namespace,
		CertManagerIssuer: clusterConfig.CertManagerIssuer,
		ServiceName:       clusterConfig.ServiceName,
		ServicePort:       clusterConfig.ServicePort,
	}

	return err
}

// SetTLS creates or updates a TLS secret in the Kubernetes cluster for the provided vanity domain.
func (c *KubeClient) SetTLS(ctx context.Context, job jobs.VanityDomain) error {
	secret := &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("%s-tls-cert", safeDomainName(job.VanityDomain)),
			Namespace: c.Namespace,
			Labels: map[string]string{
				"Domain":   job.VanityDomain,
				"Provided": "true",
			},
		},
		Type: v1.SecretTypeTLS,

		Data: map[string][]byte{
			"ca.crt":  []byte(""), // Nothing for now
			"tls.key": []byte(job.ProvidedCertificate.Key),
			"tls.crt": []byte(job.ProvidedCertificate.Cert),
		},
	}

	existingSecret, err := c.client.CoreV1().Secrets(c.Namespace).Get(ctx, fmt.Sprintf("%s-tls-cert", safeDomainName(job.VanityDomain)), metaV1.GetOptions{})
	if err != nil {
		log.Println(err)
	}

	if err != nil && strings.Contains(err.Error(), "not found") {
		_, err := c.client.CoreV1().Secrets(c.Namespace).Create(ctx, secret, metaV1.CreateOptions{})
		if err != nil {
			return err
		}
	} else {
		existingSecret.ObjectMeta.Annotations = secret.ObjectMeta.Annotations
		existingSecret.Labels = secret.Labels
		existingSecret.Data = secret.Data

		if _, err = c.client.CoreV1().Secrets(c.Namespace).Update(ctx, secret, metaV1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// UnSetTLS deletes a TLS secret in the Kubernetes cluster for the provided vanity domain.
func (c *KubeClient) UnSetTLS(ctx context.Context, job jobs.VanityDomain) error {
	err := c.client.CoreV1().Secrets(c.Namespace).Delete(ctx, fmt.Sprintf("%s-tls-cert", safeDomainName(job.VanityDomain)), metaV1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}

	return nil
}

// SetVanityDomain creates or updates an Ingress resource in the Kubernetes cluster for the provided vanity domain.
func (c *KubeClient) SetVanityDomain(ctx context.Context, job jobs.VanityDomain) error {

	var pathType networkingV1.PathType = networkingV1.PathTypeImplementationSpecific
	ingress := &networkingV1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        safeDomainName(job.VanityDomain),
			Namespace:   c.Namespace,
			Annotations: map[string]string{},
		},
		Spec: networkingV1.IngressSpec{
			Rules: []networkingV1.IngressRule{
				{
					Host: job.VanityDomain,
					IngressRuleValue: networkingV1.IngressRuleValue{
						HTTP: &networkingV1.HTTPIngressRuleValue{
							Paths: []networkingV1.HTTPIngressPath{
								{
									PathType: &pathType,
									Backend: networkingV1.IngressBackend{
										Service: &networkingV1.IngressServiceBackend{
											Name: c.ServiceName,
											Port: networkingV1.ServiceBackendPort{
												Number: c.ServicePort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: []networkingV1.IngressTLS{
				{
					Hosts:      []string{job.VanityDomain},
					SecretName: fmt.Sprintf("%s-tls-cert", safeDomainName(job.VanityDomain)), // This should be created by cert-manager
				},
			},
		},
	}

	if job.ProvidedCertificate == nil && c.CertManagerIssuer != "" {
		ingress.ObjectMeta.Annotations["cert-manager.io/cluster-issuer"] = c.CertManagerIssuer
	}

	existingIngress, err := c.client.NetworkingV1().Ingresses(c.Namespace).Get(ctx, ingress.Name, metaV1.GetOptions{})
	if err != nil {
		log.Println(err)
	}

	if err != nil && strings.Contains(err.Error(), "not found") {
		if _, err := c.client.NetworkingV1().Ingresses(c.Namespace).Create(ctx, ingress, metaV1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create ingress for vanity domain %s: %v", job.VanityDomain, err)
		}
	} else {
		existingIngress.ObjectMeta.Annotations = ingress.ObjectMeta.Annotations
		existingIngress.Labels = ingress.Labels
		existingIngress.Spec = ingress.Spec

		if _, err = c.client.NetworkingV1().Ingresses(c.Namespace).Update(ctx, existingIngress, metaV1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// UnSetCustomDomain deletes an Ingress resource in the Kubernetes cluster for the provided vanity domain.
func (c *KubeClient) UnSetVanityDomain(ctx context.Context, job jobs.VanityDomain) error {
	err := c.client.NetworkingV1().Ingresses(c.Namespace).Delete(ctx, safeDomainName(job.VanityDomain), metaV1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}

	return nil
}

func safeDomainName(domain string) string {
	// Replace any invalid characters with a hyphen
	// This is a simple implementation, you might want to use a more robust validation
	return strings.ReplaceAll(domain, ".", "-")
}
