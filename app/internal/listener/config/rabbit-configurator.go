package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common"
	amqp "github.com/rabbitmq/amqp091-go"
)

var rabbitMQEnvName = "RABBITMQ_URL"
var prefetchCountEnvName = "PREFETCH_COUNT"

func ConfigureRabbit() (*RabbitConfig, error) {
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		msg := "RABBITMQ_URL env is not set"
		return nil, &common.ScanError{
			Message: msg,
			Code:    500,
		}
	}
	slog.Info("Successfully fetched env",
		slog.String("env_name", rabbitMQEnvName),
		slog.String("value", rabbitmqURL),
	)

	prefetchCountStr := os.Getenv("PREFETCH_COUNT")
	if prefetchCountStr == "" {
		msg := "PREFETCH_COUNT env is not set"
		return nil, &common.ScanError{
			Message: msg,
			Code:    500,
		}
	}
	prefetchCount, prefetchErr := strconv.Atoi(prefetchCountStr)
	if prefetchErr != nil {
		msg := fmt.Sprintf("PREFETCH_COUNT has to be a number %s", prefetchCountStr)
		return nil, &common.ScanError{
			Message: msg,
			Code:    500,
		}
	}
	slog.Info("Successfully fetched env",
		slog.String("env_name", prefetchCountEnvName),
		slog.Int("value", prefetchCount),
	)

	conn, dialErr := amqp.Dial(rabbitmqURL)
	if dialErr != nil {
		msg := fmt.Sprintf("Cannot create TCP dial connection with %s, %v", rabbitmqURL, dialErr)
		return nil, &common.ScanError{
			Message: msg,
			Code:    500,
		}
	}
	slog.Info("TCP dial connection created")

	errMidConnChann := conn.NotifyClose(make(chan *amqp.Error))
	taskChannel, connErr := conn.Channel()
	if connErr != nil {
		msg := fmt.Sprintf("Error during opening the Channel with %s, %v", rabbitmqURL, connErr)
		return nil, &common.ScanError{
			Message: msg,
			Code:    500,
		}
	}
	slog.Info("Channel created", slog.String("url", rabbitmqURL))

	qosErr := taskChannel.Qos(
		prefetchCount,
		0,
		false,
	)
	if qosErr != nil {
		msg := fmt.Sprintf("Error with setting PrefetchCount on RabbitMQ config, %v", qosErr)
		return nil, &common.ScanError{
			Message: msg,
			Code:    500,
		}
	}
	slog.Info("PrefetchCount property set", slog.Int("prefetch_count", prefetchCount))

	return &RabbitConfig{
		ConnCh:       conn,
		TaskCh:       taskChannel,
		ErrMidConnCh: errMidConnChann,
	}, nil
}
