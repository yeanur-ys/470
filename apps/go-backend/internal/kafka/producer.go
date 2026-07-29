package kafka

import (
	"context"
	"encoding/json"

	kafkago "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:     kafkago.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafkago.LeastBytes{},
		},
	}
}

// Publish emits a derived, already-processed event (e.g. "claim resolved",
// "auditor slashed") for other services (frontend real-time layer, analytics)
// to subscribe to. It's separate from the raw Debezium CDC topics.
func (p *Producer) Publish(ctx context.Context, key string, event any) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafkago.Message{Key: []byte(key), Value: body})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
