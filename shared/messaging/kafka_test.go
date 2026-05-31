package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

func TestConsumerHandleRecordSuccess(t *testing.T) {
	t.Parallel()

	state := &consumerTestState{}
	consumer := newTestConsumer(state, func(context.Context, EventEnvelope) error {
		state.handlerCalls++
		return nil
	})

	record := newTestRecord(t, EventEnvelope{
		EventID:       "evt-1",
		EventType:     "OrderConfirmed",
		AggregateID:   "ord-1",
		CorrelationID: "corr-1",
		Payload:       map[string]any{"status": "CONFIRMED"},
	})

	if err := consumer.handleRecord(context.Background(), record); err != nil {
		t.Fatalf("handleRecord error: %v", err)
	}
	if state.handlerCalls != 1 {
		t.Fatalf("expected 1 handler call, got %d", state.handlerCalls)
	}
	if state.commitCalls != 1 {
		t.Fatalf("expected 1 commit, got %d", state.commitCalls)
	}
	if len(state.published) != 0 {
		t.Fatalf("expected no dlq publish, got %d", len(state.published))
	}
}

func TestConsumerHandleRecordRetryThenSuccess(t *testing.T) {
	t.Parallel()

	state := &consumerTestState{}
	consumer := newTestConsumer(state, func(context.Context, EventEnvelope) error {
		state.handlerCalls++
		if state.handlerCalls < 3 {
			return errors.New("temporary db error")
		}
		return nil
	})
	consumer.options.MaxRetries = 3
	consumer.options.InitialBackoff = 100 * time.Millisecond

	record := newTestRecord(t, EventEnvelope{
		EventID:       "evt-2",
		EventType:     "OrderCancelled",
		AggregateID:   "ord-2",
		CorrelationID: "corr-2",
		Payload:       map[string]any{"status": "CANCELLED"},
	})

	if err := consumer.handleRecord(context.Background(), record); err != nil {
		t.Fatalf("handleRecord error: %v", err)
	}
	if state.handlerCalls != 3 {
		t.Fatalf("expected 3 handler calls, got %d", state.handlerCalls)
	}
	if len(state.sleeps) != 2 {
		t.Fatalf("expected 2 backoff sleeps, got %d", len(state.sleeps))
	}
	if state.sleeps[0] != 100*time.Millisecond || state.sleeps[1] != 200*time.Millisecond {
		t.Fatalf("unexpected backoff sequence: %#v", state.sleeps)
	}
	if state.commitCalls != 1 {
		t.Fatalf("expected 1 commit, got %d", state.commitCalls)
	}
}

func TestConsumerHandleRecordPublishDLQAfterRetryLimit(t *testing.T) {
	t.Parallel()

	state := &consumerTestState{}
	consumer := newTestConsumer(state, func(context.Context, EventEnvelope) error {
		state.handlerCalls++
		return errors.New("postgres timeout")
	})
	consumer.options.MaxRetries = 2
	consumer.options.InitialBackoff = 50 * time.Millisecond

	record := newTestRecord(t, EventEnvelope{
		EventID:       "evt-3",
		EventType:     "OrderConfirmed",
		AggregateID:   "ord-3",
		CorrelationID: "corr-3",
		Payload:       map[string]any{"status": "CONFIRMED"},
	})
	record.Partition = 2
	record.Offset = 15

	if err := consumer.handleRecord(context.Background(), record); err != nil {
		t.Fatalf("handleRecord error: %v", err)
	}
	if state.handlerCalls != 3 {
		t.Fatalf("expected 3 handler calls, got %d", state.handlerCalls)
	}
	if state.commitCalls != 1 {
		t.Fatalf("expected commit after dlq, got %d", state.commitCalls)
	}
	if len(state.published) != 1 {
		t.Fatalf("expected 1 dlq publish, got %d", len(state.published))
	}
	if got := state.published[0].Topic; got != "order.events.dlq" {
		t.Fatalf("unexpected dlq topic: %s", got)
	}

	var dlq DLQMessage
	if err := json.Unmarshal(state.published[0].Value, &dlq); err != nil {
		t.Fatalf("unmarshal dlq payload: %v", err)
	}
	if dlq.OriginalTopic != "order.events" || dlq.OriginalOffset != 15 || dlq.FailedService != "test-service" {
		t.Fatalf("unexpected dlq payload: %+v", dlq)
	}
	if dlq.OriginalPayload == "" {
		t.Fatal("expected original payload in dlq")
	}
}

func TestConsumerHandleRecordPoisonMessageToDLQ(t *testing.T) {
	t.Parallel()

	state := &consumerTestState{}
	consumer := newTestConsumer(state, func(context.Context, EventEnvelope) error {
		state.handlerCalls++
		return nil
	})

	record := &kgo.Record{
		Topic: "order.events",
		Key:   []byte("ord-4"),
		Value: []byte(`{"broken_json"`),
		Headers: []kgo.RecordHeader{
			{Key: "x-correlation-id", Value: []byte("corr-4")},
			{Key: "x-event-id", Value: []byte("evt-4")},
			{Key: "x-event-type", Value: []byte("OrderConfirmed")},
		},
	}

	if err := consumer.handleRecord(context.Background(), record); err != nil {
		t.Fatalf("handleRecord error: %v", err)
	}
	if state.handlerCalls != 0 {
		t.Fatalf("expected handler not called, got %d", state.handlerCalls)
	}
	if state.commitCalls != 1 {
		t.Fatalf("expected commit after poison dlq, got %d", state.commitCalls)
	}
	if len(state.published) != 1 {
		t.Fatalf("expected 1 dlq publish, got %d", len(state.published))
	}

	var dlq DLQMessage
	if err := json.Unmarshal(state.published[0].Value, &dlq); err != nil {
		t.Fatalf("unmarshal dlq payload: %v", err)
	}
	if dlq.OriginalPayload != `{"broken_json"` {
		t.Fatalf("unexpected poison payload in dlq: %q", dlq.OriginalPayload)
	}
}

func TestConsumerHandleRecordNonRetryableGoesDirectToDLQ(t *testing.T) {
	t.Parallel()

	state := &consumerTestState{}
	consumer := newTestConsumer(state, func(context.Context, EventEnvelope) error {
		state.handlerCalls++
		return MarkNonRetryable(errors.New("unsupported event payload"))
	})
	consumer.options.MaxRetries = 5

	record := newTestRecord(t, EventEnvelope{
		EventID:       "evt-5",
		EventType:     "OrderConfirmed",
		AggregateID:   "ord-5",
		CorrelationID: "corr-5",
		Payload:       map[string]any{"status": "CONFIRMED"},
	})

	if err := consumer.handleRecord(context.Background(), record); err != nil {
		t.Fatalf("handleRecord error: %v", err)
	}
	if state.handlerCalls != 1 {
		t.Fatalf("expected 1 handler call, got %d", state.handlerCalls)
	}
	if len(state.sleeps) != 0 {
		t.Fatalf("expected no retry sleep, got %#v", state.sleeps)
	}
	if len(state.published) != 1 {
		t.Fatalf("expected direct dlq publish, got %d", len(state.published))
	}
}

type consumerTestState struct {
	handlerCalls int
	commitCalls  int
	published    []*kgo.Record
	sleeps       []time.Duration
}

func newTestConsumer(state *consumerTestState, handler EventHandler) *Consumer {
	return &Consumer{
		handler: handler,
		options: normalizeConsumerOptions(ConsumerOptions{
			ServiceName:    "test-service",
			MaxRetries:     1,
			InitialBackoff: 25 * time.Millisecond,
			DLQSuffix:      ".dlq",
		}),
		commitRecords: func(context.Context, ...*kgo.Record) error {
			state.commitCalls++
			return nil
		},
		publishRecord: func(_ context.Context, record *kgo.Record) error {
			state.published = append(state.published, record)
			return nil
		},
		sleep: func(duration time.Duration) {
			state.sleeps = append(state.sleeps, duration)
		},
	}
}

func newTestRecord(t *testing.T, envelope EventEnvelope) *kgo.Record {
	t.Helper()

	payload, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return &kgo.Record{
		Topic: "order.events",
		Key:   []byte(envelope.AggregateID),
		Value: payload,
		Headers: []kgo.RecordHeader{
			{Key: "x-correlation-id", Value: []byte(envelope.CorrelationID)},
			{Key: "x-event-id", Value: []byte(envelope.EventID)},
			{Key: "x-event-type", Value: []byte(envelope.EventType)},
		},
	}
}
