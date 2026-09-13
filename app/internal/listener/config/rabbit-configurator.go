package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	rabbitMQEnvName      = "RABBITMQ_URL"
	prefetchCountEnvName = "PREFETCH_COUNT"
	queueNameEnv         = "RABBITMQ_QUEUE_NAME"

	prefetchCountDefaultValue = "12"
)

func ConfigureRabbit() (*RabbitConfig, error) {
	rabbitmqURL := os.Getenv(rabbitMQEnvName)
	if rabbitmqURL == "" {
		msg := "RABBITMQ_URL env is not set"
		return nil, &types.ScanError{
			Message: msg,
			Code:    500,
		}
	}
	slog.Info("Successfully fetched env",
		slog.String("env_name", rabbitMQEnvName),
		slog.String("value", rabbitmqURL),
	)

	prefetchCountStr := os.Getenv(prefetchCountEnvName)
	if prefetchCountStr == "" {
		slog.Warn("PREFETCH_COUNT env is not set, fallback to default")
		prefetchCountStr = prefetchCountDefaultValue
	}
	prefetchCount, prefetchErr := strconv.Atoi(prefetchCountStr)
	if prefetchErr != nil {
		msg := fmt.Sprintf("PREFETCH_COUNT has to be a number %s", prefetchCountStr)
		return nil, &types.ScanError{
			Message: msg,
			Code:    500,
			Err:     prefetchErr,
		}
	}

	queueName := os.Getenv(queueNameEnv)
	if queueName == "" {
		msg := "RABBITMQ_QUEUE_NAME env is not set"
		return nil, &types.ScanError{
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
		return nil, &types.ScanError{
			Message: msg,
			Code:    500,
			Err:     dialErr,
		}
	}
	slog.Info("TCP dial connection created")

	errMidConnChann := conn.NotifyClose(make(chan *amqp.Error))
	taskChannel, connErr := conn.Channel()
	if connErr != nil {
		_ = conn.Close()
		msg := fmt.Sprintf("Error during opening the Channel with %s, %v", rabbitmqURL, connErr)
		return nil, &types.ScanError{
			Message: msg,
			Code:    500,
			Err:     connErr,
		}
	}
	slog.Info("Channel created", slog.String("url", rabbitmqURL))

	qosErr := taskChannel.Qos(
		prefetchCount,
		0,
		false,
	)
	if qosErr != nil {
		_ = taskChannel.Close()
		_ = conn.Close()
		msg := fmt.Sprintf("Error with setting PrefetchCount on RabbitMQ config, %v", qosErr)
		return nil, &types.ScanError{
			Message: msg,
			Code:    500,
			Err:     qosErr,
		}
	}
	slog.Info("PrefetchCount property set",
		slog.Int("prefetch_count", prefetchCount))

	return &RabbitConfig{
		ConnCh:       conn,
		TaskCh:       taskChannel,
		ErrMidConnCh: errMidConnChann,
		QueueName:    queueName,
	}, nil
}
