package worker

import (
	"errors"
	"testing"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockOrchestrator struct {
	mock.Mock
}

func (m *MockOrchestrator) Orchestrate(taskWrapper types.TaskWrapper, workerId int) error {
	args := m.Called(taskWrapper, workerId)
	return args.Error(0)
}

type MockAcknowledger struct {
	mock.Mock
}

func (m *MockAcknowledger) Ack() error {
	return m.Called().Error(0)
}

func (m *MockAcknowledger) Reject(requeue bool) error {
	return m.Called(requeue).Error(0)
}

var errGeneric = errors.New("some standard error")
var domainError = &types.ScanError{
	Message:  "Domain specific failure",
	Code:     400,
	Err:      errors.New("internal cause"),
	TaskId:   "task-123",
	WorkerId: 7,
}

var listenerTests = []struct {
	name             string
	task             types.TaskWrapper
	orchestrateErr   error
	expectedErrCount int
	expectedError    *types.ScanError
	setupAckMock     func(m *MockAcknowledger)
}{
	{
		name: "Happy path, no error sent to channel",
		task: types.TaskWrapper{
			ScanTask: types.ScanTask{TaskId: "task-123"},
		},
		orchestrateErr:   nil,
		expectedErrCount: 0,
		setupAckMock: func(m *MockAcknowledger) {
			m.On("Ack").Return(nil).Once()
		},
	},
	{
		name: "Should pass through domain ScanError",
		task: types.TaskWrapper{
			ScanTask: types.ScanTask{TaskId: "task-123"},
		},
		orchestrateErr:   domainError,
		expectedErrCount: 1,
		expectedError:    domainError,
		setupAckMock: func(m *MockAcknowledger) {
			m.On("Reject", true).Return(nil).Once()
		},
	},
	{
		name: "Should wrap unknown generic error",
		task: types.TaskWrapper{
			ScanTask: types.ScanTask{TaskId: "task-123"},
		},
		orchestrateErr:   errGeneric,
		expectedErrCount: 1,
		expectedError: &types.ScanError{
			Message:   "Unknown error",
			Code:      500,
			Err:       errGeneric,
			TaskId:    "task-123",
			WorkerId:  7,
			Component: "scan-listener",
		},
		setupAckMock: func(m *MockAcknowledger) {
			m.On("Reject", false).Return(nil).Once()
		},
	},
}

func TestScanListener_Listen(t *testing.T) {
	workerId := 7

	for _, tt := range listenerTests {
		t.Run(tt.name, func(t *testing.T) {
			mockOrchestrator := new(MockOrchestrator)
			mockOrchestrator.On("Orchestrate", mock.Anything, workerId).Return(tt.orchestrateErr).Once()

			mockAck := new(MockAcknowledger)
			if tt.setupAckMock != nil {
				tt.setupAckMock(mockAck)
			}
			tt.task.Ack = mockAck

			taskChan := make(chan types.TaskWrapper, 1)
			errChan := make(chan types.ScanError, 1)

			listener := &ScanListener{
				TaskChannel:  taskChan,
				ErrorChannel: errChan,
				Orchestrator: mockOrchestrator,
			}

			taskChan <- tt.task
			close(taskChan)

			listener.Listen(workerId)

			require.Len(t, errChan, tt.expectedErrCount)
			if tt.expectedErrCount > 0 {
				actualErr := <-errChan
				require.Equal(t, *tt.expectedError, actualErr)
			}

			mockOrchestrator.AssertExpectations(t)
			mockAck.AssertExpectations(t)
		})
	}
}

func TestScanListener_Listen_MultipleTasks(t *testing.T) {
	workerId := 7
	mockOrchestrator := new(MockOrchestrator)
	mockAck := new(MockAcknowledger)

	task1 := types.TaskWrapper{ScanTask: types.ScanTask{TaskId: "task-1"}, Ack: mockAck}
	task2 := types.TaskWrapper{ScanTask: types.ScanTask{TaskId: "task-2"}, Ack: mockAck}
	task3 := types.TaskWrapper{ScanTask: types.ScanTask{TaskId: "task-3"}, Ack: mockAck}

	task2DomainError := &types.ScanError{
		Message:  "Domain specific failure",
		Code:     400,
		Err:      errors.New("internal cause"),
		TaskId:   "task-2",
		WorkerId: workerId,
	}

	mockOrchestrator.On("Orchestrate", task1, workerId).Return(nil).Once()
	mockOrchestrator.On("Orchestrate", task2, workerId).Return(task2DomainError).Once()
	mockOrchestrator.On("Orchestrate", task3, workerId).Return(errGeneric).Once()

	mockAck.On("Ack").Return(nil).Once()
	mockAck.On("Reject", false).Return(nil).Once()
	mockAck.On("Reject", true).Return(nil).Once()

	taskChan := make(chan types.TaskWrapper, 3)
	errChan := make(chan types.ScanError, 3)

	listener := &ScanListener{
		TaskChannel:  taskChan,
		ErrorChannel: errChan,
		Orchestrator: mockOrchestrator,
	}

	taskChan <- task1
	taskChan <- task2
	taskChan <- task3

	close(taskChan)

	listener.Listen(workerId)

	require.Len(t, errChan, 2)

	err1 := <-errChan
	require.Equal(t, 400, err1.Code)
	require.Equal(t, "task-2", err1.TaskId)

	err2 := <-errChan
	require.Equal(t, 500, err2.Code)
	require.Equal(t, "task-3", err2.TaskId)

	mockOrchestrator.AssertExpectations(t)
	mockAck.AssertExpectations(t)
}
