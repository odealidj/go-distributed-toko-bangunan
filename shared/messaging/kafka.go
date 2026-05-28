package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	TopicOrderEvents     = "order.events"
	TopicInventoryEvents = "inventory.events"
	TopicPaymentEvents   = "payment.events"
)

type Producer interface {
	Publish(ctx context.Context, topic string, key string, envelope EventEnvelope) error
	Close()
}

type KgoProducer struct {
	client *kgo.Client
}

func NewKgoProducer(brokers string) (*KgoProducer, error) {
	client, err := kgo.NewClient(kgo.SeedBrokers(splitBrokers(brokers)...))
	if err != nil {
		return nil, err
	}
	return &KgoProducer{client: client}, nil
}

func (p *KgoProducer) Publish(ctx context.Context, topic string, key string, envelope EventEnvelope) error {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
		Headers: []kgo.RecordHeader{
			{Key: "x-correlation-id", Value: []byte(envelope.CorrelationID)},
			{Key: "x-causation-id", Value: []byte(envelope.CausationID)},
			{Key: "x-event-id", Value: []byte(envelope.EventID)},
			{Key: "x-event-type", Value: []byte(envelope.EventType)},
		},
	}
	if envelope.Traceparent != "" {
		record.Headers = append(record.Headers, kgo.RecordHeader{Key: "traceparent", Value: []byte(envelope.Traceparent)})
	}
	return p.client.ProduceSync(ctx, record).FirstErr()
}

func (p *KgoProducer) Close() {
	p.client.Close()
}

type EventHandler func(context.Context, EventEnvelope) error

type Consumer struct {
	client  *kgo.Client
	handler EventHandler
}

func NewConsumer(brokers string, topic string, group string, handler EventHandler) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(splitBrokers(brokers)...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, err
	}
	return &Consumer{client: client, handler: handler}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		fetches := c.client.PollFetches(ctx)
		if err := fetches.Err(); err != nil {
			if errors.Is(err, context.Canceled) || fetches.IsClientClosed() {
				return nil
			}
			return err
		}

		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			var envelope EventEnvelope
			if err := json.Unmarshal(record.Value, &envelope); err != nil {
				return err
			}
			if err := c.handler(ctx, envelope); err != nil {
				return err
			}
			if err := c.client.CommitRecords(ctx, record); err != nil {
				return err
			}
		}
	}
}

func (c *Consumer) Close() {
	c.client.Close()
}

func TopicForAggregate(aggregateType string) string {
	switch aggregateType {
	case "order":
		return TopicOrderEvents
	case "inventory", "stock_reservation":
		return TopicInventoryEvents
	case "payment":
		return TopicPaymentEvents
	default:
		return aggregateType + ".events"
	}
}

func splitBrokers(brokers string) []string {
	raw := strings.Split(brokers, ",")
	result := make([]string, 0, len(raw))
	for _, broker := range raw {
		broker = strings.TrimSpace(broker)
		if broker != "" {
			result = append(result, broker)
		}
	}
	if len(result) == 0 {
		return []string{"localhost:29092"}
	}
	return result
}
