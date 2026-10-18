package listener

import (
	"context"
	"log/slog"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/listener/config"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/router"
)

func Listen(rabbitConfig *config.RabbitConfig, ctx context.Context, taskRouter router.TaskRouter) error {
	conn := rabbitConfig.ConnCh
	taskChannel := rabbitConfig.ConsumerChannel
	errMidConnCh := rabbitConfig.ErrMidConnCh
	taskCh, taskErr := taskChannel.Consume(rabbitConfig.QueueName, "", false, false, false, false, nil)

	if taskErr != nil {
		return taskErr
	}
	slog.Info("Listening on RabbitMQ started")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping listener, received stop signal")
			if taskChannel != nil {
				_ = taskChannel.Close()
			}
			if conn != nil {
				_ = conn.Close()
			}
			return nil

		case closeMidConn := <-errMidConnCh:
			msg := "Stopping listener, error during connection occurred"
			slog.Error(msg, slog.Any("Error", closeMidConn))
			if taskChannel != nil {
				_ = taskChannel.Close()
			}
			if conn != nil {
				_ = conn.Close()
			}
			return &types.ScanError{
				Message: msg,
				Code:    500,
				Err:     closeMidConn,
			}

		case task, ok := <-taskCh:
			if !ok {
				slog.Info("RabbitMQ task channel closed")
				return nil
			}

			routerErr := taskRouter.Route(task)
			if routerErr != nil {
				slog.Warn("Message rejected", slog.String("MessageId", task.MessageId))
				_ = task.Reject(false)
			}
		}
	}
}
