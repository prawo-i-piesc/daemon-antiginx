package router

import (
	"encoding/json"
	"log/slog"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	"github.com/rabbitmq/amqp091-go"
)

type TaskRouter struct {
	ScanTaskChannel chan types.TaskWrapper
}

func (r *TaskRouter) Route(task amqp091.Delivery) error {
	var scanTask types.ScanTask

	parsingErr := json.Unmarshal(task.Body, &task)
	if parsingErr != nil {
		msg := "Scan task parsing error"
		slog.Warn(msg,
			slog.Any("error", parsingErr))
		return &types.ScanError{
			Message: msg,
			Code:    500,
			Err:     parsingErr,
		}
	}

	wrapper := types.TaskWrapper{
		QueueTask: task,
		ScanTask:  scanTask,
	}

	r.ScanTaskChannel <- wrapper
	return nil
}
