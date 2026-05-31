package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// sqsAPI is the subset of the SQS client used by SQSPublisher.
type sqsAPI interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

// SQSPublisher publishes metric events to an SQS queue.
type SQSPublisher struct {
	client   sqsAPI
	queueURL string
}

// NewSQSPublisher returns an SQSPublisher targeting queueURL.
func NewSQSPublisher(client sqsAPI, queueURL string) *SQSPublisher {
	return &SQSPublisher{client: client, queueURL: queueURL}
}

// retryDelays are the wait durations between attempts (3 total attempts = 2 retries).
var retryDelays = []time.Duration{50 * time.Millisecond, 200 * time.Millisecond, 800 * time.Millisecond}

// Publish marshals events to JSON and sends them to SQS with up to 3 attempts.
// On final failure the full payload is logged to slog for Loki recovery.
func (p *SQSPublisher) Publish(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	body, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("metrics: marshal events: %w", err)
	}
	msg := string(body)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelays[attempt-1]):
			}
		}
		_, lastErr = p.client.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:    aws.String(p.queueURL),
			MessageBody: aws.String(msg),
		})
		if lastErr == nil {
			return nil
		}
	}

	// Final failure: log payload for Loki-based manual recovery.
	slog.ErrorContext(ctx, "metrics: failed to publish to SQS after 3 attempts",
		"error", lastErr,
		"queue_url", p.queueURL,
		"payload", msg,
	)
	return fmt.Errorf("metrics: SQS publish failed after 3 attempts: %w", lastErr)
}
