package main

import (
	"context"

	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

// ProduceMessages consumes the in channel and sends the message
func ProduceMessages(brokers string, topic string, async bool, events *chan []byte) {
	go func() {
		w := kafka.NewWriter(kafka.WriterConfig{
			Brokers:  []string{brokers},
			Topic:    topic,
			Balancer: &kafka.Hash{},
			Async:    async,
		})

		defer w.Close()

		for v := range *events {
			go func(v []byte) {
				err := w.WriteMessages(context.Background(),
					kafka.Message{
						Key:   nil,
						Value: v,
					},
				)
				if err != nil {
					log.Errorf("message write failed; will try again: %v", err)
					*events <- v
					return
				}
			}(v)
		}
	}()
}
