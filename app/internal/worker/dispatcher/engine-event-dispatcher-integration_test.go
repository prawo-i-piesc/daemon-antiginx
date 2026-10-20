//go:build e2e

package dispatcher

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	eventhandle "github.com/prawo-i-piesc/daemon-antiginx/app/internal/worker/dispatcher/event-handle"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockResultPublisher struct {
	mock.Mock
}

func (m *MockResultPublisher) Publish(ctx context.Context, taskId, target string, payload []byte) error {
	args := m.Called(ctx, taskId, target, payload)
	return args.Error(0)
}

type MockMetricsFacadeIntegration struct {
	mock.Mock
}

func (m *MockMetricsFacadeIntegration) AddCounter(metricName string, value float64, labels ...string) {
	m.Called(metricName, value, labels)
}

func (m *MockMetricsFacadeIntegration) ObserveHistogram(metricName string, value float64, labels ...string) {
	m.Called(metricName, value, labels)
}

func (m *MockMetricsFacadeIntegration) SetGauge(metricName string, value float64, labels ...string) {
	m.Called(metricName, value, labels)
}

func TestIntegration_DispatcherToServicesFlow(t *testing.T) {
	taskId := "int-task-999"
	target := "integration-target.com"
	workerId := 1

	mockPublisher := new(MockResultPublisher)
	mockPublisher.On("Publish", mock.Anything, taskId, target, mock.Anything).Return(nil).Once()

	mockMetrics := new(MockMetricsFacadeIntegration)
	mockMetrics.On("AddCounter", "test_counter", 1.0, []string{"prod"}).Return().Once()

	resultService := &eventhandle.ResultServiceImpl{
		Context:     context.Background(),
		MaxRetries:  1,
		BackoffTime: 1,
		Publisher:   mockPublisher,
	}

	handler := &eventhandle.EventsHandlerImpl{
		ResultService:  resultService,
		MetricsService: mockMetrics,
	}

	errorChannel := make(chan types.ScanError, 5)
	dispatcher := NewEngineEventDispatcher(errorChannel, handler)

	eventStream := make(chan eventhandle.EngineEventWrapper, 2)

	resultPayload := []byte(`{
		"target": "integration-target.com",
		"testId": "int-task-999",
		"result": {
			"Name": "nmap-scan",
			"Certainty": 100,
			"ThreatLevel": 4,
			"Description": "Integration Test"
		},
		"endFlag": false,
		"resultType": 1
	}`)

	metricsPayload := []byte(`{
		"Name": "test_counter",
		"Type": 1,
		"Labels": ["prod"],
		"Value": 1.0
	}`)

	eventStream <- eventhandle.EngineEventWrapper{
		EventType: eventhandle.Result,
		Event:     json.RawMessage(resultPayload),
	}

	eventStream <- eventhandle.EngineEventWrapper{
		EventType: eventhandle.Metrics,
		Event:     json.RawMessage(metricsPayload),
	}

	close(eventStream)

	dispatcher.Dispatch(eventStream, taskId, workerId, target)

	require.Len(t, errorChannel, 0)
	mockPublisher.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}
