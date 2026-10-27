package event_handle

import (
	"context"
	"log/slog"
	"time"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ResultPublisher interface {
	Publish(ctx context.Context, taskId, target string, payload []byte) error
}

type RabbitMQClient struct {
	Conn        *amqp.Connection
	QueueName   string
	BackoffTime int
}

func (r *RabbitMQClient) Publish(ctx context.Context, taskId string, target string, payload []byte) error {
	channel, confirmation, err := r.prepareChannel(r.Conn, taskId, target)
	if err != nil {
		return err
	}
	defer channel.Close()

	pubCtx, cancel := context.WithTimeout(ctx, time.Duration(r.BackoffTime)*time.Millisecond)
	defer cancel()

	err = channel.PublishWithContext(
		pubCtx,
		"",
		r.QueueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         payload,
		},
	)
	if err != nil {
		return err
	}

	select {
	case <-pubCtx.Done():
		return pubCtx.Err()
	case conf, ok := <-confirmation:
		if !ok {
			return &types.ScanError{
				Message:   "Rabbit closed channel before confirmation",
				Code:      500,
				Err:       nil,
				TaskId:    taskId,
				Component: "result-publisher",
			}
		}
		if !conf.Ack {
			return &types.ScanError{
				Message:   "Message rejected by broker",
				Code:      500,
				Err:       nil,
				TaskId:    taskId,
				Component: "result-publisher",
			}
		}
	}

	slog.Info("Successfully published scan results",
		slog.String("TaskId", taskId),
		slog.String("Target", target))
	return nil
}

func (r *RabbitMQClient) prepareChannel(conn *amqp.Connection, taskId string, target string) (*amqp.Channel, <-chan amqp.Confirmation, error) {
	oneTimeChannel, chanErr := conn.Channel()
	if chanErr != nil {
		slog.Error("Failed to open channel for publisher",
			slog.String("TaskId", taskId),
			slog.String("Target", target))
		return nil, nil, chanErr
	}

	if err := oneTimeChannel.Confirm(false); err != nil {
		slog.Error("Failed to enable Confirmation mode",
			slog.String("TaskId", taskId),
			slog.String("Target", target))
		_ = oneTimeChannel.Close()
		return nil, nil, err
	}

	confirms := oneTimeChannel.NotifyPublish(make(chan amqp.Confirmation, 1))
	return oneTimeChannel, confirms, nil
}
