package worker

import (
	"errors"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
)

type Listener interface {
	Listen(workerId int)
}
type ScanListener struct {
	TaskChannel  chan types.TaskWrapper
	ErrorChannel chan types.ScanError
	Orchestrator Orchestrator
}

func (l *ScanListener) Listen(workerId int) {
	for taskWrapper := range l.TaskChannel {
		orchestrationErr := l.Orchestrator.Orchestrate(taskWrapper, workerId)

		var orchestrationErrorDomain *types.ScanError
		if orchestrationErr != nil {
			if errors.As(orchestrationErr, &orchestrationErrorDomain) {
				_ = taskWrapper.Ack.Reject(true)
				l.ErrorChannel <- *orchestrationErrorDomain
			} else {
				_ = taskWrapper.Ack.Reject(false)
				l.ErrorChannel <- types.ScanError{
					Message:   "Unknown error",
					Code:      500,
					Err:       orchestrationErr,
					TaskId:    taskWrapper.ScanTask.TaskId,
					WorkerId:  workerId,
					Component: "scan-listener",
				}
			}
		} else {
			_ = taskWrapper.Ack.Ack()
		}
	}
}
