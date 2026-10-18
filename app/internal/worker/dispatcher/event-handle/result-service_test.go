package event_handle

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	methodName = "Publish"
	target     = "example.com"
	taskId     = "task-123"
)

var testEvent = ResultEvent{
	Target: target,
	TestId: taskId,
	Result: TestResult{
		Name:        "nmap-scan",
		Certainty:   100,
		ThreatLevel: High,
		Metadata:    map[string]string{"port": "443"},
		Description: "Znaleziono otwarty port HTTPS",
	},
	EndFlag:    false,
	ResultType: Success,
	ProcessInfo: RequestInfo{
		Message: "Zakończono test pomyślnie",
		Code:    200,
	},
}

var badEvent = ResultEvent{
	Result: TestResult{
		Metadata: make(chan int),
	},
}

var tests = []struct {
	name        string
	maxRetries  int
	cancelCtx   bool
	setupMock   func(m *MockResultPublisher)
	assert      func(t *testing.T, m *MockResultPublisher)
	resultEvent ResultEvent
}{
	{
		name:       "Happy Path, without retry",
		maxRetries: 2,
		setupMock: func(m *MockResultPublisher) {
			m.On(methodName, mock.Anything, taskId, target, mock.Anything).
				Return(nil).Once()
		},
		assert:      assertResultEvent,
		resultEvent: testEvent,
	},
	{
		name:       "Happy Path, retry once",
		maxRetries: 2,
		setupMock: func(m *MockResultPublisher) {
			m.On(methodName, mock.Anything, taskId, target, mock.Anything).
				Return(errors.New("timeout")).Once()
			m.On(methodName, mock.Anything, taskId, target, mock.Anything).
				Return(nil).Once()
		},
		assert:      assertResultEvent,
		resultEvent: testEvent,
	},
	{
		name:       "Should exhaust all retries",
		maxRetries: 2,
		setupMock: func(m *MockResultPublisher) {
			m.On(methodName, mock.Anything, taskId, target, mock.Anything).
				Return(errors.New("fatal rabbit error")).Times(3)
		},
		assert:      assertResultEvent,
		resultEvent: testEvent,
	},
	{
		name:       "Should continue publishing despite context cancel",
		maxRetries: 2,
		cancelCtx:  true,
		setupMock: func(m *MockResultPublisher) {
			m.On(methodName, mock.Anything, taskId, target, mock.Anything).
				Return(errors.New("some error")).Times(3)
		},
		assert:      assertResultEvent,
		resultEvent: testEvent,
	},
	{
		name:       "Should skip event when marshall function return error",
		maxRetries: 2,
		cancelCtx:  false,
		assert: func(t *testing.T, m *MockResultPublisher) {
			m.AssertNotCalled(t, methodName)
		},
		resultEvent: badEvent,
	},
}

type MockResultPublisher struct {
	mock.Mock
}

func (m *MockResultPublisher) Publish(ctx context.Context, taskId, target string, payload []byte) error {
	args := m.Called(ctx, taskId, target, payload)
	return args.Error(0)
}

func TestResultService_publishResults(t *testing.T) {

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPub := new(MockResultPublisher)
			if tt.setupMock != nil {
				tt.setupMock(mockPub)
			}

			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancelCtx {
				cancel()
			} else {
				defer cancel()
			}

			service := &ResultServiceImpl{
				Context:     ctx,
				MaxRetries:  tt.maxRetries,
				BackoffTime: 1,
				Publisher:   mockPub,
			}

			service.publishResults(target, taskId, tt.resultEvent)

			mockPub.AssertExpectations(t)

			if tt.assert != nil {
				tt.assert(t, mockPub)
			}
		})
	}
}

func assertResultEvent(t *testing.T, m *MockResultPublisher) {
	if assert.NotEmpty(t, m.Calls) {
		payload := m.Calls[0].Arguments.Get(3).([]byte)
		assertPayloadContents(t, payload, target, taskId)
	}
}
func assertPayloadContents(t *testing.T, payload []byte, expectedTarget, expectedTaskId string) {
	t.Helper()

	var got ScanContext
	err := json.Unmarshal(payload, &got)
	require.NoError(t, err)

	assert.Equal(t, expectedTarget, got.Target)
	assert.Equal(t, expectedTaskId, got.TestId)

	assert.Equal(t, Success, got.Result.ResultType)
	assert.Equal(t, "nmap-scan", got.Result.TestResult.Name)
	assert.Equal(t, High, got.Result.TestResult.ThreatLevel)

	assert.Equal(t, 200, got.Result.ScanInfo.Code)
	assert.Equal(t, "Zakończono test pomyślnie", got.Result.ScanInfo.Message)
}
