package listener

import (
	"context"
	"log/slog"

	"github.com/prawo-i-piesc/scan-daemon/app/internal/common/types"
	"github.com/prawo-i-piesc/scan-daemon/app/internal/listener/config"
	"github.com/prawo-i-piesc/scan-daemon/app/internal/router"
)

func Listen(rabbitConfig *config.RabbitConfig, ctx context.Context, taskRouter router.TaskRouter) error {
	conn := rabbitConfig.ConnCh
	taskChannel := rabbitConfig.TaskCh
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
			_ = taskChannel.Close()
			_ = conn.Close()
			return nil

		case closeMidConn := <-errMidConnCh:
			msg := "Stopping listener, error during connection occurred"
			slog.Error(msg,
				slog.Any("Error", closeMidConn))
			_ = taskChannel.Close()
			_ = conn.Close()
			return &types.ScanError{
				Message: msg,
				Code:    500,
				Err:     closeMidConn,
			}

		case task := <-taskCh:
			routerErr := taskRouter.Route(task)
			return routerErr
		}
	}

}
