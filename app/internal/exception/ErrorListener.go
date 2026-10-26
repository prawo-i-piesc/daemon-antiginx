package exception

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	event_handle "github.com/prawo-i-piesc/daemon-antiginx/app/internal/worker/dispatcher/event-handle"
)

type ErrorListener struct {
	ErrorChannel   chan types.ScanError
	CancelFunc     context.CancelFunc
	MetricsService event_handle.MetricsFacade
}

func (l *ErrorListener) Listen() {
	slog.Info("Listening on internal error channel started")

	for err := range l.ErrorChannel {
		l.MetricsService.AddCounter(event_handle.InternalErrorsTotal, 1, strconv.Itoa(err.Code), err.Component)
		switch err.Code {
		case 500:
			slog.Error("Fatal error, application shutdown",
				slog.String("message", err.Message),
				slog.Int("code", err.Code),
				slog.String("error", err.Err.Error()),
				slog.String("taskId", err.TaskId),
				slog.Int("workerId", err.WorkerId),
				slog.String("component", err.Component))
			l.CancelFunc()

		case 400:
			slog.Warn(err.Message,
				slog.Int("code", err.Code),
				slog.String("error", err.Err.Error()),
				slog.String("taskId", err.TaskId),
				slog.Int("workerId", err.WorkerId),
				slog.String("component", err.Component),
			)
		}
	}
}
