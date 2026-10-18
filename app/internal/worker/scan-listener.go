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
				l.ErrorChannel <- *orchestrationErrorDomain
			} else {
				l.ErrorChannel <- types.ScanError{
					Message:  "Unknown error",
					Code:     500,
					Err:      orchestrationErr,
					TaskId:   taskWrapper.ScanTask.TaskId,
					WorkerId: workerId,
				}
			}
		}
	}
}
