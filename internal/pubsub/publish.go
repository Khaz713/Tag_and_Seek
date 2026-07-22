package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/Khaz713/Tag_and_Seek/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func EncodeGob[T any](v T) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func PublishGameResult(ch *amqp.Channel, result routing.GameResult) error {
	body, err := EncodeGob(result)
	if err != nil {
		return fmt.Errorf("failed to encode game result: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(
		ctx,
		"",
		routing.GameResultQueue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/octet-stream",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish game result: %w", err)
	}

	return nil
}
