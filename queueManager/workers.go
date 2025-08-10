package queueManager

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/geekgonecrazy/vanityDomainManager/jobs"
	"github.com/geekgonecrazy/vanityDomainManager/kubernetes"
	"github.com/geekgonecrazy/vanityDomainManager/verifiers"

	"github.com/nats-io/nats.go/jetstream"
)

func (q *queueManager) startDomainJobWorker() error {
	q.logger.Println("Starting Domain job Worker")

	config := jetstream.ConsumerConfig{
		Name:          "vanityDomainVerifier-domainjob-worker",
		Durable:       "vanityDomainVerifier-domainjob-worker",
		Description:   "The consumer for the vanityDomainVerifier",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: q.GetJobSubject(">"),
		MaxDeliver:    10,
	}
	con, err := q.jobStream.CreateOrUpdateConsumer(context.Background(), config)
	if err != nil {
		return fmt.Errorf("create or update consumer: %w", err)
	}

	_, err = con.Consume(q.domainJobHandler(config))
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	return nil
}

func (q *queueManager) configureVanityDomain(referenceID string, domain jobs.VanityDomain) error {
	q.logger.Printf("Configuring Vanity Domain %s ...", domain.VanityDomain)

	// should do a dns check here to see if the VanityDomain is pointing to DesiredDNSTarget with DesiredDNSTargetType
	if domain.VanityDomain == "" || (len(domain.DesiredARecordTargets) == 0 && domain.DesiredCNAMETarget == "") || domain.DesiredDNSTargetType == "" {
		return fmt.Errorf("Invalid job data for vanity domain: %s, skipping", domain.VanityDomain)
	}

	q.logger.Printf("Verifying Vanity Domain %s", domain.VanityDomain)

	if err := verifiers.VerifyDomain(domain); err != nil {
		return fmt.Errorf("Domain verification failed for %s: %s", domain.VanityDomain, err)
	}

	q.logger.Printf("Vanity Domain %s verified successfully", domain.VanityDomain)

	if domain.ProvidedCertificate != nil {
		q.logger.Printf("Validating TLS Certificate provided for %s", domain.VanityDomain)
		if err := verifiers.ValidateTLSCert(domain); err != nil {
			return fmt.Errorf("TLS certificate validation failed for %s: %s", domain.VanityDomain, err)
		}

		q.logger.Println("TLS Certificate Validated!")

		q.logger.Println("Inserting TLS Certificate into environment")

		if err := kubernetes.GetClient().SetTLS(context.Background(), domain); err != nil {
			return fmt.Errorf("Failed to set TLS for %s: %s", domain.VanityDomain, err)
		}

		q.logger.Println("TLS Certificate Ready for use!")
	}

	q.logger.Println("Setting Vanity Domain in Environment")

	if err := kubernetes.GetClient().SetVanityDomain(context.Background(), domain); err != nil {
		return fmt.Errorf("Failed to set custom domain for %s: %s", domain.VanityDomain, err)
	}

	q.logger.Println("Vanity Domain Set in Environment Successfully!")

	if err := q.SendStatusUpdate(referenceID, true, "", false); err != nil {
		return fmt.Errorf("failed to send status update for %s: %s", domain.VanityDomain, err)
	}

	return nil
}

func (q *queueManager) domainRemove(referenceID string, domain jobs.VanityDomain) error {
	q.logger.Printf("Removing Vanity Domain %s", domain.VanityDomain)

	q.logger.Printf("Removing TLS Certificate provided for %s", domain.VanityDomain)
	if err := kubernetes.GetClient().UnSetTLS(context.Background(), domain); err != nil {
		return fmt.Errorf("Failed to remove TLS for %s: %s", domain.VanityDomain, err)
	}

	q.logger.Printf("TLS Certificiate removed for %s", domain.VanityDomain)

	q.logger.Printf("Removing Vanity Domain from Environment %s", domain.VanityDomain)

	if err := kubernetes.GetClient().UnSetVanityDomain(context.Background(), domain); err != nil {
		return fmt.Errorf("Failed to remove Vanity Domain for %s: %s", domain.VanityDomain, err)
	}

	q.logger.Printf("Vanity Domain %s removed from Environment Successfully!", domain.VanityDomain)

	if err := q.SendStatusUpdate(referenceID, true, "", false); err != nil {
		return fmt.Errorf("failed to send status update for %s: %s", domain.VanityDomain, err)
	}

	return nil
}

func (q *queueManager) domainJobHandler(config jetstream.ConsumerConfig) func(msg jetstream.Msg) {
	return func(msg jetstream.Msg) {
		q.logger.Printf("Received message on subject %s", msg.Subject())

		hasError := true // Assume failure by default
		errorMsg := ""
		referenceID := "unknown"
		defer func() {
			q.ackornack(config, msg, referenceID, hasError, errorMsg)
		}()

		var job jobs.DomainJob
		if err := json.Unmarshal(msg.Data(), &job); err != nil {
			q.logger.Printf("failed to unmarshal job err: %s", err)
			return
		}

		referenceID = job.ReferenceID

		switch job.Type {
		case "add":
			q.logger.Printf("Processing Vanity Domain Add for %s", job.Domain.VanityDomain)
			if err := q.configureVanityDomain(job.ReferenceID, job.Domain); err != nil {
				errorMsg = err.Error()
				return
			}
		case "change":
			q.logger.Printf("Processing Vanity Domain Change for %s", job.Domain.VanityDomain)
			if err := q.configureVanityDomain(job.ReferenceID, job.Domain); err != nil {
				errorMsg = err.Error()
				return
			}
		case "remove":
			if err := q.domainRemove(job.ReferenceID, job.Domain); err != nil {
				errorMsg = err.Error()
				return
			}
		}

		q.logger.Println("Message Processed Successfully")
		hasError = false
	}
}
