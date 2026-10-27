package event_handle

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	resultMethodName = "publishResults"
	addCounter       = "AddCounter"
	observe          = "ObserveHistogram"
	setGauge         = "SetGauge"
)

var testEventPayload = []byte(`{
	"target": "example.com",
	"testId": "task-123",
	"result": {
		"Name": "nmap-scan",
		"Certainty": 100,
		"ThreatLevel": 4,
		"Metadata": {
			"port": "443"
		},
		"Description": "Znaleziono otwarty port HTTPS"
	},
	"endFlag": false,
	"resultType": "Success",
	"message": {
		"Message": "Zakończono test pomyślnie",
		"Code": 200
	}
}`)

type MockResultService struct {
	mock.Mock
}

type MockMetricsFacade struct {
	mock.Mock
}

func (m *MockResultService) publishResults(target string, taskId string, result ResultEvent) {
	m.Called(target, taskId, result)
}

func (m *MockMetricsFacade) ObserveHistogram(metricName string, value float64, labels ...string) {
	m.Called(metricName, value, labels)
}

func (m *MockMetricsFacade) AddCounter(metricName string, value float64, labels ...string) {
	m.Called(metricName, value, labels)
}

func (m *MockMetricsFacade) SetGauge(metricName string, value float64, labels ...string) {
	m.Called(metricName, value, labels)
}

func (m *MockMetricsFacade) IncGauge(metricName string, labels ...string) {
	m.Called(metricName, labels)
}
func (m *MockMetricsFacade) DecGauge(metricName string, labels ...string) {
	m.Called(metricName, labels)
}

type HandleResultTest struct {
	name         string
	eventPayload []byte
	expectError  bool
	setupMock    func(m *MockResultService)
	assert       func(t *testing.T, m *MockResultService)
}

type HandleMetricsTest struct {
	name         string
	eventPayload []byte
	expectError  bool
	setupMock    func(m *MockMetricsFacade)
	assert       func(t *testing.T, m *MockMetricsFacade)
}

var handleResultsTest = []HandleResultTest{
	{
		name:         "Happy path, event published",
		eventPayload: testEventPayload,
		expectError:  false,
		setupMock: func(m *MockResultService) {
			m.On("publishResults", target, taskId, mock.AnythingOfType("ResultEvent")).Once()
		},
		assert: func(t *testing.T, m *MockResultService) {
			m.AssertCalled(t, resultMethodName, target, taskId, mock.AnythingOfType("ResultEvent"))
		},
	},
	{
		name:         "Should return error when parsing error occurs",
		eventPayload: []byte(`{"target": "example.com"`),
		expectError:  true,
		setupMock: func(m *MockResultService) {
		},
		assert: func(t *testing.T, m *MockResultService) {
			m.AssertNotCalled(t, resultMethodName)
		},
	},
	{
		name:         "Should return nil when endFlag occurs",
		eventPayload: []byte(`{"endFlag":true}`),
		expectError:  false,
		setupMock: func(m *MockResultService) {
		},
		assert: func(t *testing.T, m *MockResultService) {
			m.AssertNotCalled(t, resultMethodName)
		},
	},
}

var metricTests = []HandleMetricsTest{
	{
		name:         "Happy path, register Counter",
		eventPayload: []byte(`{"Name":"test_counter","Type":1,"Labels":["prod","eu"],"Value":10.5}`),
		expectError:  false,
		setupMock: func(m *MockMetricsFacade) {
			m.On(addCounter, "test_counter", 10.5, []string{"prod", "eu"}).Once()
		},
		assert: func(t *testing.T, m *MockMetricsFacade) {
			m.AssertCalled(t, addCounter, "test_counter", 10.5, []string{"prod", "eu"})
		},
	},
	{
		name:         "Happy path, register Histogram",
		eventPayload: []byte(`{"Name":"test_hist","Type":0,"Labels":["GET"],"Value":42.0}`),
		expectError:  false,
		setupMock: func(m *MockMetricsFacade) {
			m.On(observe, "test_hist", 42.0, []string{"GET"}).Once()
		},
		assert: func(t *testing.T, m *MockMetricsFacade) {
			m.AssertCalled(t, observe, "test_hist", 42.0, []string{"GET"})
		},
	},
	{
		name:         "Happy path, register Gauge",
		eventPayload: []byte(`{"Name":"test_gauge","Type":2,"Labels":[],"Value":5.0}`),
		expectError:  false,
		setupMock: func(m *MockMetricsFacade) {
			m.On(setGauge, "test_gauge", 5.0, []string{}).Once()
		},
		assert: func(t *testing.T, m *MockMetricsFacade) {
			m.AssertCalled(t, setGauge, "test_gauge", 5.0, []string{})
		},
	},
	{
		name:         "Should return error when parsing error occurs",
		eventPayload: []byte(`{"Name": "test_counter", "Type": 1`),
		expectError:  true,
		setupMock: func(m *MockMetricsFacade) {
		},
		assert: func(t *testing.T, m *MockMetricsFacade) {
			m.AssertNotCalled(t, addCounter)
			m.AssertNotCalled(t, observe)
			m.AssertNotCalled(t, setGauge)
		},
	},
}

func TestEventsHandler_HandleResult(t *testing.T) {
	for _, tt := range handleResultsTest {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockResultService)
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			handler := &EventsHandlerImpl{
				ResultService: mockService,
			}

			err := handler.HandleResult(tt.eventPayload, taskId, target)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			mockService.AssertExpectations(t)

			if tt.assert != nil {
				tt.assert(t, mockService)
			}
		})
	}
}

func TestEventsHandler_HandleMetrics(t *testing.T) {
	for _, tt := range metricTests {
		t.Run(tt.name, func(t *testing.T) {
			mockMetrics := new(MockMetricsFacade)
			if tt.setupMock != nil {
				tt.setupMock(mockMetrics)
			}

			handler := &EventsHandlerImpl{
				MetricsService: mockMetrics,
			}

			err := handler.HandleMetrics(tt.eventPayload, taskId, target)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			mockMetrics.AssertExpectations(t)

			if tt.assert != nil {
				tt.assert(t, mockMetrics)
			}
		})
	}
}

func TestEventsHandler_HandleLog(t *testing.T) {
	var tests = []struct {
		name           string
		eventPayload   []byte
		taskId         string
		target         string
		expectError    bool
		expectedLogMsg string
	}{
		{
			name:           "Happy path, successfully parses and logs event",
			eventPayload:   []byte(`{"Message":"Engine scanning started","TestId":"test-456","Level":"INFO"}`),
			taskId:         "task-123",
			target:         "example.com",
			expectError:    false,
			expectedLogMsg: "Engine scanning started",
		},
		{
			name:           "Should return error on invalid JSON",
			eventPayload:   []byte(`{"Message":"Broken log json"`),
			taskId:         "task-123",
			target:         "example.com",
			expectError:    true,
			expectedLogMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
			oldLogger := slog.Default()
			slog.SetDefault(logger)
			defer slog.SetDefault(oldLogger)

			handler := &EventsHandlerImpl{}
			err := handler.HandleLog(tt.eventPayload, tt.taskId, tt.target)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				logOutput := logBuffer.String()
				assert.Contains(t, logOutput, tt.expectedLogMsg)
				assert.Contains(t, logOutput, "TaskId="+tt.taskId)
				assert.Contains(t, logOutput, "Target="+tt.target)
				assert.Contains(t, logOutput, "TestId=test-456")
			}
		})
	}
}
