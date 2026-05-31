package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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

const defaultDLQSuffix = ".dlq"

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

type ConsumerOptions struct {
	ServiceName    string
	MaxRetries     int
	InitialBackoff time.Duration
	DLQSuffix      string
}

type DLQMessage struct {
	OriginalTopic     string            `json:"original_topic"`
	OriginalPartition int32             `json:"original_partition"`
	OriginalOffset    int64             `json:"original_offset"`
	OriginalKey       string            `json:"original_key,omitempty"`
	OriginalHeaders   map[string]string `json:"original_headers"`
	OriginalPayload   string            `json:"original_payload"`
	ErrorMessage      string            `json:"error_message"`
	FailedService     string            `json:"failed_service"`
	FailedAt          time.Time         `json:"failed_at"`
}

type Consumer struct {
	client        *kgo.Client
	handler       EventHandler
	options       ConsumerOptions
	commitRecords func(context.Context, ...*kgo.Record) error
	publishRecord func(context.Context, *kgo.Record) error
	sleep         func(time.Duration)
}

func NewConsumer(brokers string, topic string, group string, handler EventHandler, options ConsumerOptions) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(splitBrokers(brokers)...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, err
	}
	options = normalizeConsumerOptions(options)
	return &Consumer{
		client:  client,
		handler: handler,
		options: options,
		commitRecords: func(ctx context.Context, records ...*kgo.Record) error {
			return client.CommitRecords(ctx, records...)
		},
		publishRecord: func(ctx context.Context, record *kgo.Record) error {
			return client.ProduceSync(ctx, record).FirstErr()
		},
		sleep: time.Sleep,
	}, nil
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
			if err := c.handleRecord(ctx, record); err != nil {
				return err
			}
		}
	}
}

func (c *Consumer) Close() {
	c.client.Close()
}

func (c *Consumer) handleRecord(ctx context.Context, record *kgo.Record) error {
	carrier := kafkaHeadersCarrier(record.Headers)
	messageCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)

	var envelope EventEnvelope
	unmarshalErr := json.Unmarshal(record.Value, &envelope)
	messageCtx = observability.WithRequestScope(
		messageCtx,
		observability.NewExecutionID("kfk"),
		firstNonEmpty(envelope.CorrelationID, carrier.Get("x-correlation-id")),
	)
	messageCtx, span := kafkaTracer.Start(messageCtx, "Kafka consume "+record.Topic+" "+firstNonEmpty(envelope.EventType, carrier.Get("x-event-type")),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", record.Topic),
			attribute.String("messaging.operation", "consume"),
			attribute.String("event_id", firstNonEmpty(envelope.EventID, carrier.Get("x-event-id"))),
			attribute.String("event_type", firstNonEmpty(envelope.EventType, carrier.Get("x-event-type"))),
			attribute.String("order_id", envelope.AggregateID),
			attribute.String("correlation_id", firstNonEmpty(envelope.CorrelationID, carrier.Get("x-correlation-id"))),
		),
	)
	defer span.End()

	if unmarshalErr != nil {
		span.RecordError(unmarshalErr)
		span.SetStatus(codes.Error, unmarshalErr.Error())
		return c.sendToDLQAndCommit(messageCtx, record, unmarshalErr)
	}

	if err := c.processWithRetry(messageCtx, envelope); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return c.sendToDLQAndCommit(messageCtx, record, err)
	}
	if err := c.commitRecords(messageCtx, record); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (c *Consumer) processWithRetry(ctx context.Context, envelope EventEnvelope) error {
	var lastErr error
	for attempt := 0; attempt <= c.options.MaxRetries; attempt++ {
		if err := c.handler(ctx, envelope); err != nil {
			lastErr = err
			if isNonRetryable(err) {
				return err
			}
			if attempt == c.options.MaxRetries {
				break
			}
			c.sleep(backoffDelay(c.options.InitialBackoff, attempt))
			continue
		}
		return nil
	}
	return lastErr
}

func (c *Consumer) sendToDLQAndCommit(ctx context.Context, record *kgo.Record, err error) error {
	if publishErr := c.publishDLQ(ctx, record, err); publishErr != nil {
		return fmt.Errorf("publish dlq: %w", publishErr)
	}
	if commitErr := c.commitRecords(ctx, record); commitErr != nil {
		return fmt.Errorf("commit dlq record: %w", commitErr)
	}
	return nil
}

func (c *Consumer) publishDLQ(ctx context.Context, record *kgo.Record, err error) error {
	payload, marshalErr := json.Marshal(DLQMessage{
		OriginalTopic:     record.Topic,
		OriginalPartition: record.Partition,
		OriginalOffset:    record.Offset,
		OriginalKey:       string(record.Key),
		OriginalHeaders:   headersToMap(record.Headers),
		OriginalPayload:   string(record.Value),
		ErrorMessage:      err.Error(),
		FailedService:     c.options.ServiceName,
		FailedAt:          time.Now().UTC(),
	})
	if marshalErr != nil {
		return marshalErr
	}

	dlqRecord := &kgo.Record{
		Topic: DLQTopic(record.Topic, c.options.DLQSuffix),
		Key:   record.Key,
		Value: payload,
		Headers: []kgo.RecordHeader{
			{Key: "x-original-topic", Value: []byte(record.Topic)},
			{Key: "x-failed-service", Value: []byte(c.options.ServiceName)},
			{Key: "x-error-message", Value: []byte(err.Error())},
		},
	}
	for _, header := range record.Headers {
		dlqRecord.Headers = append(dlqRecord.Headers, header)
	}
	return c.publishRecord(ctx, dlqRecord)
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

func DLQTopic(topic string, suffix string) string {
	if suffix == "" {
		suffix = defaultDLQSuffix
	}
	return topic + suffix
}

func normalizeConsumerOptions(options ConsumerOptions) ConsumerOptions {
	if options.MaxRetries < 0 {
		options.MaxRetries = 0
	}
	if options.InitialBackoff <= 0 {
		options.InitialBackoff = 250 * time.Millisecond
	}
	if options.DLQSuffix == "" {
		options.DLQSuffix = defaultDLQSuffix
	}
	return options
}

func backoffDelay(initial time.Duration, attempt int) time.Duration {
	delay := initial
	for i := 0; i < attempt; i++ {
		delay *= 2
		if delay > 5*time.Second {
			return 5 * time.Second
		}
	}
	return delay
}

type nonRetryableError struct {
	err error
}

func (e nonRetryableError) Error() string {
	return e.err.Error()
}

func (e nonRetryableError) Unwrap() error {
	return e.err
}

func MarkNonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return nonRetryableError{err: err}
}

func isNonRetryable(err error) bool {
	var target nonRetryableError
	return errors.As(err, &target)
}

func headersToMap(headers []kgo.RecordHeader) map[string]string {
	result := make(map[string]string, len(headers))
	for _, header := range headers {
		result[header.Key] = string(header.Value)
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
