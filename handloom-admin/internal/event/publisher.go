package event

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"

	"github.com/handloom/admin/pkg/logger"
)

// EventPublisher is the interface every publisher implementation satisfies.
type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
}

// EventHandler is the callback interface used by LocalPublisher.
type EventHandler interface {
	CanHandle(eventType EventType) bool
	Handle(ctx context.Context, event Event) error
}

// ---------------------------------------------------------------------------
// SNSPublisher — publishes to an AWS SNS topic
// ---------------------------------------------------------------------------

// SNSPublisher publishes events to an AWS SNS topic with an event_type
// message attribute so subscribers can use filter policies.
type SNSPublisher struct {
	client   *sns.Client
	topicARN string
}

// NewSNSPublisher creates an SNSPublisher. When endpoint is non-empty
// (local dev / LocalStack) it uses static credentials.
func NewSNSPublisher(ctx context.Context, topicARN, region, endpoint string) (*SNSPublisher, error) {
	var cfg aws.Config
	var err error

	if endpoint != "" {
		cfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				"local", "local", "",
			)),
		)
	} else {
		cfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
		)
	}
	if err != nil {
		return nil, err
	}

	var client *sns.Client
	if endpoint != "" {
		client = sns.NewFromConfig(cfg, func(o *sns.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	} else {
		client = sns.NewFromConfig(cfg)
	}

	return &SNSPublisher{
		client:   client,
		topicARN: topicARN,
	}, nil
}

// Publish serialises the event to JSON and publishes it to the SNS topic.
func (p *SNSPublisher) Publish(ctx context.Context, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.client.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(p.topicARN),
		Message:  aws.String(string(payload)),
		MessageAttributes: map[string]snstypes.MessageAttributeValue{
			"event_type": {
				DataType:    aws.String("String"),
				StringValue: aws.String(string(event.Type)),
			},
		},
	})
	return err
}

// ---------------------------------------------------------------------------
// LocalPublisher — in-process, synchronous dispatch for monolith dev mode
// ---------------------------------------------------------------------------

// LocalPublisher dispatches events to registered EventHandler implementations
// synchronously. Handler errors are logged but not propagated (fire-and-forget).
type LocalPublisher struct {
	handlers []EventHandler
	log      *logger.Logger
}

// NewLocalPublisher creates a LocalPublisher with the given handlers.
func NewLocalPublisher(log *logger.Logger, handlers ...EventHandler) *LocalPublisher {
	return &LocalPublisher{
		handlers: handlers,
		log:      log,
	}
}

// Publish iterates over registered handlers and invokes those that can handle
// the event type. Errors are logged but never returned.
func (p *LocalPublisher) Publish(ctx context.Context, event Event) error {
	for _, h := range p.handlers {
		if h.CanHandle(event.Type) {
			if err := h.Handle(ctx, event); err != nil {
				p.log.WithError(err).WithField("event_type", string(event.Type)).
					WithField("event_id", event.ID).
					Error("local event handler failed")
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// NoopPublisher — discards all events (tests / default fallback)
// ---------------------------------------------------------------------------

// NoopPublisher silently discards every event.
type NoopPublisher struct{}

// NewNoopPublisher returns a NoopPublisher.
func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

// Publish is a no-op.
func (p *NoopPublisher) Publish(_ context.Context, _ Event) error {
	return nil
}
