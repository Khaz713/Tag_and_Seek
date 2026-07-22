package pubsub

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"

	"github.com/Khaz713/Tag_and_Seek/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func DecodeGob[T any](data []byte) (T, error) {
	d := bytes.NewBuffer(data)
	dec := gob.NewDecoder(d)
	var dat T
	err := dec.Decode(&dat)
	if err != nil {
		return dat, err
	}
	return dat, nil
}

func SubscribeGameResult(conn *amqp.Connection, handler func(result routing.GameResult) error) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open consumer channel: %w", err)
	}
	err = ch.Qos(1, 0, false)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := ch.Consume(
		routing.GameResultQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		log.Println("Ready to consume game results...")
		for d := range msgs {
			result, err := DecodeGob[routing.GameResult](d.Body)
			if err != nil {
				log.Printf("Failed to decode game result: %v. Rejecting message.", err)
				d.Nack(false, false)
				continue
			}

			err = handler(result)
			if err != nil {
				log.Printf("Failed to process game result: %v. Re-queueing message", err)
				d.Nack(false, true)
				continue
			}

			err = d.Ack(false)
			if err != nil {
				log.Printf("Failed to Ack message: %v", err)
			}
		}
	}()
	return nil
}
