package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

type ProducerConfig struct {
	Brokers         []string
	Topic           string
	RequiredAcks    sarama.RequiredAcks
	Compression     sarama.CompressionCodec
	FlushFrequency  time.Duration
	Idempotent      bool
	ReturnSuccesses bool
	Version         sarama.KafkaVersion
}

type ConsumerConfig struct {
	Brokers           []string
	Topic             string
	GroupID           string
	Version           sarama.KafkaVersion
	InitialOffset     int64
	RebalanceStrategy sarama.BalanceStrategy
}

type syncProducer interface {
	SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error)
	Close() error
}

type consumerGroup interface {
	Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error
	Close() error
}

type Producer struct {
	producer syncProducer
	topic    string
}

type Consumer struct {
	group consumerGroup
	topic string
}

func NewProducer(cfg ProducerConfig) (*Producer, error) {
	if err := validateBrokers(cfg.Brokers); err != nil {
		return nil, err
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("kafka topic cannot be empty")
	}

	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.RequiredAcks = cfg.RequiredAcks
	producerConfig.Producer.Compression = cfg.Compression
	producerConfig.Producer.Flush.Frequency = cfg.FlushFrequency
	producerConfig.Producer.Idempotent = cfg.Idempotent
	producerConfig.Producer.Return.Successes = true

	if cfg.Idempotent {
		producerConfig.Net.MaxOpenRequests = 1
		producerConfig.Producer.Retry.Max = 3
	}
	if cfg.Version != (sarama.KafkaVersion{}) {
		producerConfig.Version = cfg.Version
	}

	producer, err := sarama.NewSyncProducer(cfg.Brokers, producerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	return NewProducerWithSyncProducer(cfg.Topic, producer), nil
}

func NewProducerWithSyncProducer(topic string, producer syncProducer) *Producer {
	return &Producer{producer: producer, topic: topic}
}

func (p *Producer) Publish(_ context.Context, key, value []byte, headers map[string]string) error {
	message := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}

	for headerKey, headerValue := range headers {
		message.Headers = append(message.Headers, sarama.RecordHeader{
			Key:   []byte(headerKey),
			Value: []byte(headerValue),
		})
	}

	return p.PublishMessage(message)
}

func (p *Producer) PublishMessage(message *sarama.ProducerMessage) error {
	if p.producer == nil {
		return fmt.Errorf("kafka producer is not initialized")
	}
	if message == nil {
		return fmt.Errorf("kafka message cannot be nil")
	}
	if message.Topic == "" {
		message.Topic = p.topic
	}
	if message.Topic == "" {
		return fmt.Errorf("kafka topic cannot be empty")
	}

	if _, _, err := p.producer.SendMessage(message); err != nil {
		return fmt.Errorf("failed to publish kafka message: %w", err)
	}

	return nil
}

func (p *Producer) Close() error {
	if p.producer == nil {
		return nil
	}
	return p.producer.Close()
}

func NewConsumer(cfg ConsumerConfig) (*Consumer, error) {
	if err := validateBrokers(cfg.Brokers); err != nil {
		return nil, err
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("kafka topic cannot be empty")
	}
	if cfg.GroupID == "" {
		return nil, fmt.Errorf("kafka group ID cannot be empty")
	}

	groupConfig := sarama.NewConfig()
	groupConfig.Consumer.Return.Errors = true
	groupConfig.Consumer.Offsets.Initial = cfg.InitialOffset

	if groupConfig.Consumer.Offsets.Initial == 0 {
		groupConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	}
	if cfg.RebalanceStrategy != nil {
		groupConfig.Consumer.Group.Rebalance.Strategy = cfg.RebalanceStrategy
	}
	if cfg.Version != (sarama.KafkaVersion{}) {
		groupConfig.Version = cfg.Version
	}

	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, groupConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer group: %w", err)
	}

	return NewConsumerWithGroup(cfg.Topic, group), nil
}

func NewConsumerWithGroup(topic string, group consumerGroup) *Consumer {
	return &Consumer{group: group, topic: topic}
}

func (c *Consumer) Consume(ctx context.Context, handler func(context.Context, *sarama.ConsumerMessage) error) error {
	if c.group == nil {
		return fmt.Errorf("kafka consumer is not initialized")
	}
	if handler == nil {
		return fmt.Errorf("kafka handler cannot be nil")
	}

	consumerHandler := &consumerGroupHandler{ctx: ctx, handler: handler}
	if err := c.group.Consume(ctx, []string{c.topic}, consumerHandler); err != nil {
		return fmt.Errorf("failed to consume kafka messages: %w", err)
	}

	return nil
}

func (c *Consumer) Close() error {
	if c.group == nil {
		return nil
	}
	return c.group.Close()
}

type consumerGroupHandler struct {
	ctx     context.Context
	handler func(context.Context, *sarama.ConsumerMessage) error
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-h.ctx.Done():
			return h.ctx.Err()
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			if err := h.handler(h.ctx, message); err != nil {
				return err
			}
			session.MarkMessage(message, "")
		}
	}
}

func validateBrokers(brokers []string) error {
	if len(brokers) == 0 {
		return fmt.Errorf("kafka brokers cannot be empty")
	}

	for _, broker := range brokers {
		if broker == "" {
			return fmt.Errorf("kafka broker address cannot be empty")
		}
	}

	return nil
}
