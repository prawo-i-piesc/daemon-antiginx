package worker

import "github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"

type ScanListener struct {
	TaskChannel chan types.TaskWrapper
}

func (l *ScanListener) Listen(workerId int) {

}
