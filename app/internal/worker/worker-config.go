package worker

import (
	"log/slog"
	"os"
	"strconv"
	"sync"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
)

const (
	workerPoolEnvName     = "WORKER_POOL"
	defaultWorkerPoolSize = "10"
)

func ConfigureWorkers(listener Listener) (*sync.WaitGroup, error) {
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
		return nil, &types.ScanError{
			Message:   msg,
			Code:      500,
			Err:       workerPoolErr,
			Component: "worker-config",
		}
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < workerPoolValue; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			listener.Listen(workerId)
		}(i)
	}
	return wg, nil
}
