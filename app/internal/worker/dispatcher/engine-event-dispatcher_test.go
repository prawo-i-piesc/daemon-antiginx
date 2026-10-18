package dispatcher

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/worker/dispatcher/event-handle"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testTaskID   = "task-123"
	testTarget   = "example.com"
	testWorkerID = 42
)

type MockEventsHandler struct {
	mock.Mock
}

var handlerError = errors.New("handler failed")
var internalHandlerError = &types.ScanError{
	Message:  "internal handler error",
	Code:     422,
	Err:      errors.New("invalid event"),
	TaskId:   "internal-task",
	WorkerId: 7,
}

var tests = []struct {
	name          string
	eventType     event_handle.EngineEventType
	handlerMethod string
	handlerError  error
	wantError     *types.ScanError
}{
	{
		name:          "dispatches result event",
		eventType:     event_handle.Result,
		handlerMethod: "HandleResult",
	},
	{
		name:          "dispatches log event",
		eventType:     event_handle.Log,
		handlerMethod: "HandleLog",
	},
	{
		name:          "dispatches metrics event",
		eventType:     event_handle.Metrics,
		handlerMethod: "HandleMetrics",
	},
	{
		name:          "dispatches error event",
		eventType:     event_handle.Error,
		handlerMethod: "HandleError",
	},
	{
		name:          "converts regular handler error",
		eventType:     event_handle.Error,
		handlerMethod: "HandleError",
		handlerError:  handlerError,
		wantError: &types.ScanError{
			Message:  "Unknown handler error",
			Code:     500,
			Err:      handlerError,
			TaskId:   testTaskID,
			WorkerId: testWorkerID,
		},
	},
	{
		name:          "forwards internal scan error",
		eventType:     event_handle.Error,
		handlerMethod: "HandleError",
		handlerError:  internalHandlerError,
		wantError:     internalHandlerError,
	},
	{
		name:          "handles unknown event type",
		eventType:     999,
		handlerMethod: "",
		wantError: &types.ScanError{
			Message:  "Unknown event type received",
			Code:     500,
			TaskId:   testTaskID,
			WorkerId: testWorkerID,
		},
	},
}

func (m *MockEventsHandler) HandleResult(event json.RawMessage, taskId string, target string) error {
	return m.handle("HandleResult", event, taskId, target)
}

func (m *MockEventsHandler) HandleLog(event json.RawMessage, taskId string, target string) error {
	return m.handle("HandleLog", event, taskId, target)
}

func (m *MockEventsHandler) HandleMetrics(event json.RawMessage, taskId string, target string) error {
	return m.handle("HandleMetrics", event, taskId, target)
}

func (m *MockEventsHandler) HandleError(event json.RawMessage, taskId string, target string) error {
	return m.handle("HandleError", event, taskId, target)
}

func (m *MockEventsHandler) handle(method string, event json.RawMessage, taskId string, target string) error {
	args := m.MethodCalled(method, event, taskId, target)
	if err, ok := args.Get(0).(error); ok {
		return err
	}
	return nil
}

func TestEngineEventDispatcher_Dispatch(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventPayload := json.RawMessage(`{"message":"test"}`)
			handler := new(MockEventsHandler)

			if tt.handlerMethod != "" {
				handler.On(tt.handlerMethod, eventPayload, testTaskID, testTarget).
					Return(tt.handlerError).
					Once()
			}

			errorChannel := make(chan types.ScanError, 1)
			dispatcher := NewEngineEventDispatcher(errorChannel, handler)
			eventStream := make(chan event_handle.EngineEventWrapper, 1)

			eventStream <- event_handle.EngineEventWrapper{
				EventType: tt.eventType,
				Event:     eventPayload,
			}
			close(eventStream)

			dispatcher.Dispatch(eventStream, testTaskID, testWorkerID, testTarget)

			if tt.wantError != nil {
				require.Equal(t, *tt.wantError, <-errorChannel)
			} else {
				require.Len(t, errorChannel, 0)
			}
			handler.AssertExpectations(t)
		})
	}
}

func TestEngineEventDispatcher_DispatchMultipleEvents(t *testing.T) {
	eventPayload := json.RawMessage(`{"message":"test"}`)
	handler := new(MockEventsHandler)

	handler.On("HandleResult", eventPayload, testTaskID, testTarget).Return(nil).Once()
	handler.On("HandleLog", eventPayload, testTaskID, testTarget).Return(nil).Once()

	errorChannel := make(chan types.ScanError, 1)
	dispatcher := NewEngineEventDispatcher(errorChannel, handler)
	eventStream := make(chan event_handle.EngineEventWrapper, 2)

	eventStream <- event_handle.EngineEventWrapper{EventType: event_handle.Result, Event: eventPayload}
	eventStream <- event_handle.EngineEventWrapper{EventType: event_handle.Log, Event: eventPayload}
	close(eventStream)

	dispatcher.Dispatch(eventStream, testTaskID, testWorkerID, testTarget)

	require.Len(t, errorChannel, 0)
	handler.AssertExpectations(t)
}
