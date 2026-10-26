package dispatcher

import (
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/worker/dispatcher/event-handle"
)

type EventDispatcher interface {
	Dispatch(eventStream <-chan event_handle.EngineEventWrapper, taskId string, workerId int, target string)
}

type EventHandler func(event json.RawMessage, taskId string, target string) error

type EngineEventDispatcher struct {
	handlers     map[event_handle.EngineEventType]EventHandler
	errorChannel chan types.ScanError
}

func NewEngineEventDispatcher(errorChannel chan types.ScanError, handler event_handle.EventsHandler) EngineEventDispatcher {
	d := EngineEventDispatcher{
		handlers:     make(map[event_handle.EngineEventType]EventHandler),
		errorChannel: errorChannel,
	}

	d.handlers[event_handle.Result] = handler.HandleResult
	d.handlers[event_handle.Log] = handler.HandleLog
	d.handlers[event_handle.Error] = handler.HandleError
	d.handlers[event_handle.Metrics] = handler.HandleMetrics

	return d
}

func (d *EngineEventDispatcher) Dispatch(eventStream <-chan event_handle.EngineEventWrapper, taskId string, workerId int, target string) {
	for event := range eventStream {
		handler, exists := d.handlers[event.EventType]
		if !exists {
			slog.Warn("Cannot process event, event type not handled",
				slog.Any("message", event.EventType))
			d.errorChannel <- types.ScanError{
				Message:   "Unknown event type received",
				Code:      500,
				TaskId:    taskId,
				WorkerId:  workerId,
				Component: "event-dispatcher",
			}
			continue
		}

		handlerErr := handler(event.Event, taskId, target)

		var internalHandlerErr *types.ScanError

		if handlerErr != nil && errors.As(handlerErr, &internalHandlerErr) {
			d.errorChannel <- *internalHandlerErr
		} else if handlerErr != nil {
			d.errorChannel <- types.ScanError{
				Message:   "Unknown handler error",
				Code:      500,
				Err:       handlerErr,
				TaskId:    taskId,
				WorkerId:  workerId,
				Component: "event-dispatcher",
			}
		}
	}
}
