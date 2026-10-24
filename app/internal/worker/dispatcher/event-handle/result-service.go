package event_handle

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
)

type ResultService interface {
	publishResults(target string, taskId string, result ResultEvent)
}

type ResultServiceImpl struct {
	//Register     EventRegister
	BackoffTime  int
	MaxRetries   int
	ErrorChannel chan types.ScanError
	Context      context.Context
	Publisher    ResultPublisher
}

func (r *ResultServiceImpl) publishResults(target string, taskId string, result ResultEvent) {

	scanData := ScanContext{
		Target: target,
		TestId: taskId,
		Result: ScanResult{
			TestResult: result.Result,
			ScanInfo:   result.ProcessInfo,
			ResultType: result.ResultType,
		},
		//Results: r.Register.GetResults(),
	}

	payload, err := json.Marshal(scanData)
	if err != nil {
		slog.Error("Failed to marshal scan result", slog.String("TaskId", taskId), slog.Any("Error", err))
		return
	}

	baseDelay := time.Duration(r.BackoffTime) * time.Millisecond
	safeCtx := context.WithoutCancel(r.Context)

	for attempt := 0; attempt <= r.MaxRetries; attempt++ {
		err = r.Publisher.Publish(safeCtx, taskId, target, payload)
		if err == nil {
			return
		}

		if attempt == r.MaxRetries {
			break
		}

		delay := baseDelay * time.Duration(1<<attempt)
		slog.Debug("Failed to publish result to queue, retrying...",
			slog.String("TaskId", taskId),
			slog.String("Target", target),
			slog.Int("Attempt", attempt+1),
			slog.Any("Error", err))

		select {
		case <-r.Context.Done():
			if attempt == r.MaxRetries {
				return
			}
			continue
		case <-time.After(delay):
		}
	}

	slog.Error("Exhausted all retries for publishing results",
		slog.String("TaskId", taskId),
		slog.String("Target", target))
}
