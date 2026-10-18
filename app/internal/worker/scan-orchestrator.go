package worker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"os/exec"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/worker/dispatcher"
)

type ScanOrchestrator struct {
	EngineCall      string
	EngineArg       string
	EventDispatcher dispatcher.EngineEventDispatcher
}

func (o *ScanOrchestrator) Orchestrate(taskWrapper types.TaskWrapper, workerId int) error {

	eventChan := make(chan dispatcher.EngineEventWrapper, 50)
	go o.EventDispatcher.Dispatch(eventChan, taskWrapper.ScanTask.TaskId, workerId)

	slog.Info("Processing scan...",
		slog.Int("WorkerId", workerId),
		slog.String("TaskId", taskWrapper.ScanTask.TaskId))

	cmd := exec.Command(o.EngineCall, o.EngineArg)
	cmd.Stdin = bytes.NewReader(taskWrapper.QueueTask.Body)

	stdout, stdoutErr := cmd.StdoutPipe()
	if stdoutErr != nil {
		return &types.ScanError{
			Message:  "Cannot open stdout channel",
			Code:     500,
			Err:      stdoutErr,
			TaskId:   taskWrapper.ScanTask.TaskId,
			WorkerId: workerId,
		}
	}

	if err := cmd.Start(); err != nil {
		return &types.ScanError{
			Message:  "Cannot start engine",
			Code:     500,
			Err:      err,
			TaskId:   taskWrapper.ScanTask.TaskId,
			WorkerId: workerId}
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()

		var event dispatcher.EngineEventWrapper
		if err := json.Unmarshal(line, &event); err != nil {
			slog.Warn("Faulty engine event, cannot parse (skipped)",
				slog.Any("err", err))
			continue
		}
		eventChan <- event
	}
	close(eventChan)

	if err := cmd.Wait(); err != nil {
		return &types.ScanError{
			Message:  "Engine finished with error",
			Code:     400,
			Err:      err,
			TaskId:   taskWrapper.ScanTask.TaskId,
			WorkerId: workerId,
		}
	}
	return nil
}
