package main

import (
	"context"

	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

// ProducerConfig configures the producer with needed kafka information
type ProducerConfig struct {
	Brokers []string
	Topic   string
	Async   bool
}

// Producer consumes the in channel and sends the message
// to the writer in the goroutine
func Producer(config *ProducerConfig, msgChannel chan []byte) {
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  config.Brokers,
		Topic:    config.Topic,
		Balancer: &kafka.Hash{},
		Async:    config.Async,
	})

	defer w.Close()

	for v := range msgChannel {
		go func(v []byte) {
			err := w.WriteMessages(context.Background(),
				kafka.Message{
					Key:   nil,
					Value: v,
				},
			)
			if err != nil {
				log.Error("error while writing, putting message back into the channel")
				log.Error(err)
				msgChannel <- v
				return
			}
		}(v)
	}
}
