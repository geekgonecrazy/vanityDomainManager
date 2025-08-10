package verifiers

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/geekgonecrazy/vanityDomainManager/jobs"
)

// DecodeCertPEM decodes a PEM block into a raw DER byte slice.
func DecodeCertPEM(certPEM []byte) ([]byte, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	return block.Bytes, nil
}

// ParseCertificate parses DER-encoded certificate bytes into an x509.Certificate struct.
func ParseCertificate(certDER []byte) (*x509.Certificate, error) {
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	return cert, nil
}

// ValidateTLSCert performs a complete validation of a TLS certificate.
func ValidateTLSCert(domain jobs.VanityDomain) error {
	certPEM := []byte(domain.ProvidedCertificate.Cert) // Assuming is a PEM-encoded certificate string

	certDER, err := DecodeCertPEM(certPEM)
	if err != nil {
		return err
	}

	cert, err := ParseCertificate(certDER)
	if err != nil {
		return err
	}

	// 1. Check Not Before/Not After dates
	if time.Now().Before(cert.NotBefore) {
		return errors.New("certificate is not yet valid")
	}
	if time.Now().After(cert.NotAfter) {
		return errors.New("certificate has expired")
	}

	// 2. Load system's trusted root CAs
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return fmt.Errorf("failed to get system cert pool: %w", err)
	}

	// 3. Configure and run the comprehensive verification
	opts := x509.VerifyOptions{
		DNSName:     domain.VanityDomain,
		Roots:       rootCAs,
		CurrentTime: time.Now(),
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	return nil
}
