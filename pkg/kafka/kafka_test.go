package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/IBM/sarama"
)

type fakeProducer struct {
	messages []*sarama.ProducerMessage
	sendErr  error
	closed   bool
}

func (f *fakeProducer) SendMessage(msg *sarama.ProducerMessage) (int32, int64, error) {
	if f.sendErr != nil {
		return 0, 0, f.sendErr
	}
	f.messages = append(f.messages, msg)
	return 1, 1, nil
}

func (f *fakeProducer) Close() error {
	f.closed = true
	return nil
}

type fakeConsumerGroup struct {
	err          error
	closed       bool
	consumeCalls int
	topics       []string
	messages     []*sarama.ConsumerMessage
}

func (f *fakeConsumerGroup) Consume(_ context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	f.consumeCalls++
	f.topics = append([]string(nil), topics...)
	if f.err != nil {
		return f.err
	}

	session := &fakeConsumerGroupSession{}
	claim := &fakeConsumerGroupClaim{messages: f.messages}

	if err := handler.Setup(session); err != nil {
		return err
	}
	if err := handler.ConsumeClaim(session, claim); err != nil {
		return err
	}
	return handler.Cleanup(session)
}

func (f *fakeConsumerGroup) Close() error {
	f.closed = true
	return nil
}

type fakeConsumerGroupSession struct {
	marked []*sarama.ConsumerMessage
}

func (f *fakeConsumerGroupSession) Claims() map[string][]int32 { return nil }
func (f *fakeConsumerGroupSession) MemberID() string           { return "" }
func (f *fakeConsumerGroupSession) GenerationID() int32        { return 0 }
func (f *fakeConsumerGroupSession) MarkOffset(string, int32, int64, string) {
}
func (f *fakeConsumerGroupSession) Commit() {
}
func (f *fakeConsumerGroupSession) ResetOffset(string, int32, int64, string) {
}
func (f *fakeConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, _ string) {
	f.marked = append(f.marked, msg)
}
func (f *fakeConsumerGroupSession) Context() context.Context { return context.Background() }

type fakeConsumerGroupClaim struct {
	messages []*sarama.ConsumerMessage
	ch       chan *sarama.ConsumerMessage
}

func (f *fakeConsumerGroupClaim) Topic() string              { return "events" }
func (f *fakeConsumerGroupClaim) Partition() int32          { return 0 }
func (f *fakeConsumerGroupClaim) InitialOffset() int64      { return 0 }
func (f *fakeConsumerGroupClaim) HighWaterMarkOffset() int64 { return 0 }
func (f *fakeConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	if f.ch == nil {
		f.ch = make(chan *sarama.ConsumerMessage, len(f.messages))
		for _, message := range f.messages {
			f.ch <- message
		}
		close(f.ch)
	}
	return f.ch
}

func TestNewProducerValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  ProducerConfig
	}{
		{
			name: "missing brokers",
			cfg:  ProducerConfig{Topic: "events"},
		},
		{
			name: "missing topic",
			cfg:  ProducerConfig{Brokers: []string{"localhost:9092"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			producer, err := NewProducer(tt.cfg)
			if err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if producer != nil {
				t.Fatalf("expected nil producer on validation error")
			}
		})
	}
}

func TestProducerPublish(t *testing.T) {
	producerImpl := &fakeProducer{}
	producer := NewProducerWithSyncProducer("events", producerImpl)

	err := producer.Publish(context.Background(), []byte("key"), []byte("value"), map[string]string{
		"event-type": "created",
	})
	if err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}

	if len(producerImpl.messages) != 1 {
		t.Fatalf("expected one message to be written, got %d", len(producerImpl.messages))
	}

	message := producerImpl.messages[0]
	key, err := message.Key.Encode()
	if err != nil {
		t.Fatalf("expected key to encode, got %v", err)
	}
	value, err := message.Value.Encode()
	if err != nil {
		t.Fatalf("expected value to encode, got %v", err)
	}
	if string(key) != "key" || string(value) != "value" {
		t.Fatalf("unexpected message payload: %+v", message)
	}
	if len(message.Headers) != 1 || string(message.Headers[0].Key) != "event-type" || string(message.Headers[0].Value) != "created" {
		t.Fatalf("unexpected message headers: %+v", message.Headers)
	}
}

func TestProducerPublishError(t *testing.T) {
	producerImpl := &fakeProducer{sendErr: errors.New("send failed")}
	producer := NewProducerWithSyncProducer("events", producerImpl)

	err := producer.PublishMessage(&sarama.ProducerMessage{
		Value: sarama.ByteEncoder([]byte("value")),
	})
	if err == nil {
		t.Fatalf("expected publish error, got nil")
	}
}

func TestConsumerValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  ConsumerConfig
	}{
		{
			name: "missing brokers",
			cfg: ConsumerConfig{
				Topic:   "events",
				GroupID: "group-1",
			},
		},
		{
			name: "missing topic",
			cfg: ConsumerConfig{
				Brokers: []string{"localhost:9092"},
				GroupID: "group-1",
			},
		},
		{
			name: "missing group ID",
			cfg: ConsumerConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "events",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer, err := NewConsumer(tt.cfg)
			if err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if consumer != nil {
				t.Fatalf("expected nil consumer on validation error")
			}
		})
	}
}

func TestConsumerConsume(t *testing.T) {
	group := &fakeConsumerGroup{
		messages: []*sarama.ConsumerMessage{
			{Topic: "events", Key: []byte("key"), Value: []byte("value")},
		},
	}
	consumer := NewConsumerWithGroup("events", group)

	var consumed []*sarama.ConsumerMessage
	err := consumer.Consume(context.Background(), func(_ context.Context, message *sarama.ConsumerMessage) error {
		consumed = append(consumed, message)
		return nil
	})
	if err != nil {
		t.Fatalf("expected consume to succeed, got %v", err)
	}

	if group.consumeCalls != 1 {
		t.Fatalf("expected consume to be called once, got %d", group.consumeCalls)
	}
	if len(group.topics) != 1 || group.topics[0] != "events" {
		t.Fatalf("unexpected topics: %+v", group.topics)
	}
	if len(consumed) != 1 || string(consumed[0].Key) != "key" || string(consumed[0].Value) != "value" {
		t.Fatalf("unexpected consumed messages: %+v", consumed)
	}
}

func TestConsumerConsumeError(t *testing.T) {
	group := &fakeConsumerGroup{err: errors.New("consume failed")}
	consumer := NewConsumerWithGroup("events", group)

	err := consumer.Consume(context.Background(), func(_ context.Context, message *sarama.ConsumerMessage) error {
		return nil
	})
	if err == nil {
		t.Fatalf("expected consume error, got nil")
	}
}

func TestCloseHelpers(t *testing.T) {
	producerImpl := &fakeProducer{}
	group := &fakeConsumerGroup{}

	producer := NewProducerWithSyncProducer("events", producerImpl)
	consumer := NewConsumerWithGroup("events", group)

	if err := producer.Close(); err != nil {
		t.Fatalf("expected producer close to succeed, got %v", err)
	}
	if err := consumer.Close(); err != nil {
		t.Fatalf("expected consumer close to succeed, got %v", err)
	}
	if !producerImpl.closed || !group.closed {
		t.Fatalf("expected close to be delegated")
	}
}
