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

// maxSQSBodyBytes is the per-message body budget. SQS rejects bodies over
// 256 KiB; we keep headroom for the JSON array brackets and any future message
// attributes. Events are chunked so no single SendMessage exceeds it. The
// consumer decodes a JSON array per message, so chunking is transparent to it.
const maxSQSBodyBytes = 240 * 1024

// Publish marshals events to JSON and sends them to SQS, splitting into multiple
// messages so each body stays under the SQS size limit. Each message is retried
// up to 3 times; on final failure the payload is logged to slog for recovery.
func (p *SQSPublisher) Publish(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	chunks, err := chunkEvents(events)
	if err != nil {
		return fmt.Errorf("metrics: marshal events: %w", err)
	}

	for _, chunk := range chunks {
		if sendErr := p.sendChunk(ctx, chunk); sendErr != nil {
			return sendErr
		}
	}
	return nil
}

// sendChunk marshals a chunk and sends it with up to 3 attempts.
func (p *SQSPublisher) sendChunk(ctx context.Context, events []Event) error {
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

// chunkEvents splits events into groups whose marshaled JSON body stays under
// maxSQSBodyBytes. An event whose own size exceeds the budget is emitted as a
// singleton chunk (it will fail at SQS and be logged rather than silently
// dropping the whole batch).
func chunkEvents(events []Event) ([][]Event, error) {
	const overhead = 2 // surrounding "[" + "]"
	var (
		chunks  [][]Event
		current []Event
		size    = overhead
	)
	for _, ev := range events {
		b, err := json.Marshal(ev)
		if err != nil {
			return nil, err
		}
		evSize := len(b) + 1 // event + "," separator
		if len(current) > 0 && size+evSize > maxSQSBodyBytes {
			chunks = append(chunks, current)
			current = nil
			size = overhead
		}
		current = append(current, ev)
		size += evSize
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks, nil
}
