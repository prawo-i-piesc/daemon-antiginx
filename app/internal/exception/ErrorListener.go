package exception

import (
	"log/slog"
	"os"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
)

type ErrorListener struct {
	ErrorChannel chan types.ScanError
}

func (l *ErrorListener) Listen() {
	slog.Info("Listening on internal error channel started")

	for err := range l.ErrorChannel {
		switch err.Code {
		case 500:
			slog.Error("Fatal error, application shutdown",
				slog.String("message", err.Message),
				slog.Int("code", err.Code),
				slog.String("error", err.Error()),
				slog.String("taskId", err.TaskId),
				slog.Int("workerId", err.WorkerId))
			os.Exit(1)
		case 400:
			slog.Warn(err.Message,
				slog.Int("code", err.Code),
				slog.String("error", err.Error()),
				slog.String("taskId", err.TaskId),
				slog.Int("workerId", err.WorkerId),
			)
		}
	}

}
