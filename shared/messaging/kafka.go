package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/odealidj/go-distributed-toko-bangunan/shared/observability"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
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

var kafkaTracer = otel.Tracer("shared/messaging/kafka")

func NewKgoProducer(brokers string) (*KgoProducer, error) {
	client, err := kgo.NewClient(kgo.SeedBrokers(splitBrokers(brokers)...))
	if err != nil {
		return nil, err
	}
	return &KgoProducer{client: client}, nil
}

func (p *KgoProducer) Publish(ctx context.Context, topic string, key string, envelope EventEnvelope) error {
	ctx = observability.WithCorrelationID(ctx, envelope.CorrelationID)
	ctx, span := kafkaTracer.Start(ctx, "Kafka publish "+topic+" "+envelope.EventType,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", topic),
			attribute.String("messaging.operation", "publish"),
			attribute.String("event_id", envelope.EventID),
			attribute.String("event_type", envelope.EventType),
			attribute.String("order_id", envelope.AggregateID),
			attribute.String("correlation_id", envelope.CorrelationID),
		),
	)
	defer span.End()

	if envelope.TraceID == "" {
		envelope.TraceID = observability.TraceIDFromContext(ctx)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Headers: []kgo.RecordHeader{
			{Key: "x-correlation-id", Value: []byte(envelope.CorrelationID)},
			{Key: "x-causation-id", Value: []byte(envelope.CausationID)},
			{Key: "x-event-id", Value: []byte(envelope.EventID)},
			{Key: "x-event-type", Value: []byte(envelope.EventType)},
		},
	}
	carrier := kafkaRecordCarrier{record: record}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if envelope.Traceparent == "" {
		envelope.Traceparent = carrier.Get("traceparent")
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	record.Value = payload

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
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

			carrier := kafkaHeadersCarrier(record.Headers)
			messageCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)
			messageCtx = observability.WithRequestScope(
				messageCtx,
				observability.NewExecutionID("kfk"),
				firstNonEmpty(envelope.CorrelationID, carrier.Get("x-correlation-id")),
			)
			messageCtx, span := kafkaTracer.Start(messageCtx, "Kafka consume "+record.Topic+" "+envelope.EventType,
				trace.WithSpanKind(trace.SpanKindConsumer),
				trace.WithAttributes(
					attribute.String("messaging.system", "kafka"),
					attribute.String("messaging.destination.name", record.Topic),
					attribute.String("messaging.operation", "consume"),
					attribute.String("event_id", envelope.EventID),
					attribute.String("event_type", envelope.EventType),
					attribute.String("order_id", envelope.AggregateID),
					attribute.String("correlation_id", envelope.CorrelationID),
				),
			)

			if err := c.handler(messageCtx, envelope); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				span.End()
				return err
			}
			if err := c.client.CommitRecords(messageCtx, record); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				span.End()
				return err
			}
			span.End()
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

type kafkaRecordCarrier struct {
	record *kgo.Record
}

func (c kafkaRecordCarrier) Get(key string) string {
	for _, header := range c.record.Headers {
		if strings.EqualFold(header.Key, key) {
			return string(header.Value)
		}
	}
	return ""
}

func (c kafkaRecordCarrier) Set(key, value string) {
	for index, header := range c.record.Headers {
		if strings.EqualFold(header.Key, key) {
			c.record.Headers[index].Value = []byte(value)
			return
		}
	}
	c.record.Headers = append(c.record.Headers, kgo.RecordHeader{Key: key, Value: []byte(value)})
}

func (c kafkaRecordCarrier) Keys() []string {
	keys := make([]string, 0, len(c.record.Headers))
	for _, header := range c.record.Headers {
		keys = append(keys, header.Key)
	}
	return keys
}

type kafkaHeadersCarrier []kgo.RecordHeader

func (c kafkaHeadersCarrier) Get(key string) string {
	for _, header := range c {
		if strings.EqualFold(header.Key, key) {
			return string(header.Value)
		}
	}
	return ""
}

func (c kafkaHeadersCarrier) Set(key, value string) {}

func (c kafkaHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for _, header := range c {
		keys = append(keys, header.Key)
	}
	return keys
}

var (
	_ propagation.TextMapCarrier = kafkaRecordCarrier{}
	_ propagation.TextMapCarrier = kafkaHeadersCarrier{}
)

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
