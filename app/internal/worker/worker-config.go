package worker

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/prawo-i-piesc/scan-daemon/app/internal/common/types"
)

const (
	workerPoolEnvName     = "WORKER_POOL"
	defaultWorkerPoolSize = "10"
)

func ConfigureWorkers(listener ScanListener) error {
	workerPool := os.Getenv(workerPoolEnvName)
	if workerPool == "" {
		slog.Warn("WORKER_POOL env is not set, fallback to default")
		workerPool = defaultWorkerPoolSize
	}

	workerPoolValue, workerPoolErr := strconv.Atoi(workerPool)
	if workerPoolErr != nil {
		msg := "Error occurred during task channel size parsing"
		slog.Error(msg,
			slog.Any("error", workerPoolErr))
		return &types.ScanError{
			Message: msg,
			Code:    500,
			Err:     workerPoolErr,
		}
	}

	for i := range workerPoolValue - 1 {
		go listener.Listen(i)
	}
}
