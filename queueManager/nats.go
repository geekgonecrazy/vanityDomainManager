package queueManager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/geekgonecrazy/vanityDomainManager/config"
	"github.com/geekgonecrazy/vanityDomainManager/jobs"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type SubjectType string

type queueManager struct {
	js           jetstream.JetStream
	jobStream    jetstream.Stream
	statusStream jetstream.Stream
	logger       *log.Logger
}

var _queueManager *queueManager

func Start() (*queueManager, error) {
	log.Println("Invoking a new NATS Publisher")

	natsConfig := config.Config().Nats()

	if natsConfig.ConnectionString == "" {
		return nil, fmt.Errorf("nats connection string is empty")
	}

	natsOpts := []nats.Option{}

	if !config.Config().IsDevelopment() {
		log.Println("using nats JWT authentication")
		natsOpts = append(natsOpts, nats.UserJWTAndSeed(natsConfig.JWT, natsConfig.Seed))
	}

	natsOpts = append(natsOpts,
		nats.Name("Vanity Domain Verifier"),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.Timeout(15*time.Second),
		nats.ConnectHandler(func(nc *nats.Conn) {
			log.Println("Nats has successfully connected")
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Println("Nats is disconnected because:", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Println("Nats reconnected!")
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Println("Nats connection closed")
		}),
	)

	nc, err := nats.Connect(natsConfig.ConnectionString, natsOpts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("jetstream: %w", err)
	}

	_queueManager = &queueManager{
		js:     js,
		logger: log.New(os.Stdout, "[QueueManager] ", log.LstdFlags),
	}

	if err := _queueManager.ensureStreams(); err != nil {
		return nil, err
	}

	return _queueManager, nil
}

func Mgr() *queueManager {
	if _queueManager == nil {
		panic("queue manager is not initialized")
	}
	return _queueManager
}

func (q *queueManager) ensureStreams() error {
	env := config.Config().System().Environment

	statusStream, err := q.js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:        fmt.Sprintf("%s_%s", env, "vanitydomainverifier_status"),
		Description: "Queue for Status Updates coming from Vanity Domain Verifier",
		Subjects: []string{
			q.GetStatusSubject(">"),
		},
		Retention: jetstream.WorkQueuePolicy,
		MaxAge:    7 * time.Hour * 24,
		MaxMsgs:   1_000_000_000,
		MaxBytes:  4 << 20, // 4 MB
	})
	if err != nil {
		return fmt.Errorf("add stream: %w", err)
	}

	jobStream, err := q.js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:        fmt.Sprintf("%s_%s", env, "vanitydomainverifier_jobs"),
		Description: "Job Queue for Vanity Domain Verifier",
		Subjects: []string{
			q.GetJobSubject(">"),
		},
		Retention: jetstream.WorkQueuePolicy,
		MaxAge:    7 * time.Hour * 24,
		MaxMsgs:   1_000_000_000,
		MaxBytes:  4 << 20, // 4 MB
	})
	if err != nil {
		return fmt.Errorf("add stream: %w", err)
	}

	q.jobStream = jobStream
	q.statusStream = statusStream
	return nil
}

func (q *queueManager) StartWorkers() error {
	q.logger.Println("Starting Workers")

	if err := q.startDomainJobWorker(); err != nil {
		return fmt.Errorf("start domain job worker: %w", err)
	}

	return nil
}

func (q *queueManager) AddDomainJob(job jobs.DomainJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	subjectName := q.GetJobSubject(job.ReferenceID)
	if _, err := q.js.Publish(context.Background(), subjectName, data); err != nil {
		return fmt.Errorf("publish job to %s: %w", subjectName, err)
	}

	q.logger.Printf("Job published to subject %s", subjectName)

	return nil
}

func (q *queueManager) SendStatusUpdate(referenceID string, success bool, errorMessage string, dropped bool) error {
	msg := jobs.JobStatus{
		Success:      success,
		ReferenceID:  referenceID,
		ErrorMessage: errorMessage,
		Dropped:      dropped,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("json marshal status update: %w", err)
	}

	subjectName := q.GetStatusSubject(referenceID)
	if _, err := q.js.Publish(context.Background(), subjectName, data); err != nil {
		return fmt.Errorf("publish status update to %s: %w", subjectName, err)
	}

	q.logger.Printf("Status update published to subject %s", subjectName)

	return nil
}

func (q *queueManager) GetJobSubject(sub string) string {
	return fmt.Sprintf("%s.vanityDomainVerifier.domainjob.%s", config.Config().System().Environment, sub)
}

func (q *queueManager) GetStatusSubject(sub string) string {
	return fmt.Sprintf("%s.vanityDomainVerifier.status.%s", config.Config().System().Environment, sub)
}

func (q *queueManager) ackornack(config jetstream.ConsumerConfig, msg jetstream.Msg, referenceID string, hasError bool, errorMessage string) {
	if hasError {
		// Get message info for retry logic
		msgInfo, _ := msg.Metadata()
		deliveryCount := msgInfo.NumDelivered

		q.logger.Printf("Error: %s", errorMessage)

		if deliveryCount >= uint64(config.MaxDeliver) {
			q.logger.Printf("Message %s has reached max retries (%d). Remove from the queue.", msg.Subject(), config.MaxDeliver)

			if err := q.SendStatusUpdate(referenceID, false, errorMessage, true); err != nil {
				q.logger.Printf("Failed to send status update for job %s: %s", msg.Subject(), err)
			}

			msg.Ack()
			return
		}

		if err := q.SendStatusUpdate(referenceID, false, errorMessage, false); err != nil {
			q.logger.Printf("Failed to send status update for job %s: %s", msg.Subject(), err)
		}

		// Calculate and apply exponential backoff
		baseDelay := 30 * time.Second
		nakDelay := baseDelay * time.Duration(1<<(deliveryCount-1))
		q.logger.Printf("Message %s will be retried after backoff delay of %v.", msg.Subject(), nakDelay)
		msg.NakWithDelay(nakDelay)
	} else {
		// If processing was successful, acknowledge the message
		msg.Ack()
	}
}
