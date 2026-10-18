package worker

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	event_handle "github.com/prawo-i-piesc/daemon-antiginx/app/internal/worker/dispatcher/event-handle"
	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockEventDispatcher struct {
	mock.Mock
}

func (m *MockEventDispatcher) Dispatch(eventStream <-chan event_handle.EngineEventWrapper, taskId string, workerId int, target string) {
	m.Called(eventStream, taskId, workerId, target)
	for range eventStream {
	}
}

var tests = []struct {
	name        string
	instruction string
	engineCall  string
	setupMock   func(m *MockEventDispatcher)
	expectError bool
	expectedLog string
}{
	{
		name:        "Happy path, successfully orchestrate one event",
		instruction: "SUCCESS_ONE",
		setupMock: func(m *MockEventDispatcher) {
			m.On("Dispatch", mock.Anything, "task-1", 1, "target.com").Once()
		},
		expectError: false,
	},
	{
		name:        "Happy path, successfully orchestrate multiple events",
		instruction: "SUCCESS_MULTI",
		setupMock: func(m *MockEventDispatcher) {
			m.On("Dispatch", mock.Anything, "task-1", 1, "target.com").Once()
		},
		expectError: false,
	},
	{
		name:       "Should return error when cmd Start error occurs",
		engineCall: "this-command-does-not-exist-12345",
		setupMock: func(m *MockEventDispatcher) {
			m.On("Dispatch", mock.Anything, "task-1", 1, "target.com").Once()
		},
		expectError: true,
	},
	{
		name:        "Should log warn message and skip faulty engine event",
		instruction: "FAULTY_JSON",
		setupMock: func(m *MockEventDispatcher) {
			m.On("Dispatch", mock.Anything, "task-1", 1, "target.com").Once()
		},
		expectError: false,
		expectedLog: "Faulty engine event, cannot parse (skipped)",
	},
	{
		name:        "Should return error when cmd Wait error occurs",
		instruction: "EXIT_ERROR",
		setupMock: func(m *MockEventDispatcher) {
			m.On("Dispatch", mock.Anything, "task-1", 1, "target.com").Once()
		},
		expectError: true,
	},
}

func TestScanOrchestrator_Orchestrate(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			multiWriter := io.MultiWriter(os.Stdout, &logBuffer)
			logger := slog.New(slog.NewTextHandler(multiWriter, nil))
			oldLogger := slog.Default()
			slog.SetDefault(logger)
			defer slog.SetDefault(oldLogger)

			mockDispatcher := new(MockEventDispatcher)
			if tt.setupMock != nil {
				tt.setupMock(mockDispatcher)
			}

			engineCall := os.Args[0]
			engineArg := "-test.run=TestHelperProcess"

			if tt.engineCall != "" {
				engineCall = tt.engineCall
				engineArg = ""
			}

			orchestrator := &ScanOrchestrator{
				EngineCall:      engineCall,
				EngineArg:       engineArg,
				EventDispatcher: mockDispatcher,
			}

			task := types.TaskWrapper{
				ScanTask: types.ScanTask{
					TaskId: "task-1",
					Target: "target.com",
				},
				QueueTask: amqp091.Delivery{
					Body: []byte(tt.instruction),
				},
			}

			err := orchestrator.Orchestrate(task, 1)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.expectedLog != "" {
				assert.Contains(t, logBuffer.String(), tt.expectedLog)
			}

			mockDispatcher.AssertExpectations(t)
		})
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	body, _ := io.ReadAll(os.Stdin)
	instruction := string(body)

	switch instruction {
	case "SUCCESS_ONE":
		fmt.Println(`{"EventType":0,"Event":{"message":"test1"}}`)
	case "SUCCESS_MULTI":
		fmt.Println(`{"EventType":0,"Event":{"message":"test1"}}`)
		fmt.Println(`{"EventType":0,"Event":{"message":"test2"}}`)
	case "FAULTY_JSON":
		fmt.Println(`{invalid_json_here`)
	case "EXIT_ERROR":
		os.Exit(1)
	}

	os.Exit(0)
}
