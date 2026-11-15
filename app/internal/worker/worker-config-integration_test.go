//go:build integration

package worker

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	event_handle "github.com/prawo-i-piesc/daemon-antiginx/app/internal/worker/dispatcher/event-handle"
	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type NoOpMetricsService struct{}

func (m *NoOpMetricsService) AddCounter(metricName string, value float64, labels ...string)       {}
func (m *NoOpMetricsService) ObserveHistogram(metricName string, value float64, labels ...string) {}
func (m *NoOpMetricsService) SetGauge(metricName string, value float64, labels ...string)         {}
func (m *NoOpMetricsService) IncGauge(metricName string, labels ...string)                        {}
func (m *NoOpMetricsService) DecGauge(metricName string, labels ...string)                        {}

type MockAck struct {
	mock.Mock
}

func (m *MockAck) Ack() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAck) Reject(requeue bool) error {
	args := m.Called(requeue)
	return args.Error(0)
}

type MockIntegrationDispatcher struct {
	mock.Mock
}

func (m *MockIntegrationDispatcher) Dispatch(eventStream <-chan event_handle.EngineEventWrapper, taskId string, workerId int, target string) {
	m.Called(eventStream, taskId, workerId, target)
	for range eventStream {
	}
}

func TestIntegration_WorkerListenerOrchestratorFlow(t *testing.T) {
	t.Setenv("GO_WANT_INTEGRATION_HELPER", "1")
	workerId := 10

	mockDispatcher := new(MockIntegrationDispatcher)

	mockDispatcher.On("Dispatch", mock.Anything, "int-task-ok", workerId, "target1.com").Return().Once()
	mockDispatcher.On("Dispatch", mock.Anything, "int-task-err", workerId, "target2.com").Return().Once()

	mockMetrics := &NoOpMetricsService{}

	orchestrator := &ScanOrchestrator{
		EngineCall:      os.Args[0],
		EngineArg:       "-test.run=TestIntegrationHelperProcess",
		EventDispatcher: mockDispatcher,
		MetricsService:  mockMetrics,
	}

	taskChan := make(chan types.TaskWrapper, 2)
	errChan := make(chan types.ScanError, 2)

	listener := &ScanListener{
		TaskChannel:  taskChan,
		ErrorChannel: errChan,
		Orchestrator: orchestrator,
	}

	mockAck1 := new(MockAck)
	mockAck1.On("Ack").Return(nil).Once()

	mockAck2 := new(MockAck)
	mockAck2.On("Reject", true).Return(nil).Once()

	task1 := types.TaskWrapper{
		ScanTask:  types.ScanTask{TaskId: "int-task-ok", Target: "target1.com"},
		QueueTask: amqp091.Delivery{Body: []byte("INTEGRATION_SUCCESS")},
		Ack:       mockAck1,
	}

	task2 := types.TaskWrapper{
		ScanTask:  types.ScanTask{TaskId: "int-task-err", Target: "target2.com"},
		QueueTask: amqp091.Delivery{Body: []byte("INTEGRATION_ERROR")},
		Ack:       mockAck2,
	}

	taskChan <- task1
	taskChan <- task2
	close(taskChan)

	listener.Listen(workerId)

	require.Len(t, errChan, 1)

	err := <-errChan
	require.Equal(t, 400, err.Code)
	require.Equal(t, "int-task-err", err.TaskId)

	mockDispatcher.AssertExpectations(t)
}

func TestIntegrationHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_INTEGRATION_HELPER") != "1" {
		return
	}

	body, _ := io.ReadAll(os.Stdin)
	instruction := string(body)

	switch instruction {
	case "INTEGRATION_SUCCESS":
		fmt.Println(`{"EventType":1,"Event":{"message":"integration payload ok"}}`)
		os.Exit(0)
	case "INTEGRATION_ERROR":
		os.Exit(1)
	}
}
