package event_handle

import (
	"context"
	"encoding/json"
	"log/slog"
)

type EventsHandler interface {
	HandleResult(event json.RawMessage, taskId string, target string) error
	HandleLog(event json.RawMessage, taskId string, target string) error
	HandleMetrics(event json.RawMessage, taskId string, target string) error
	HandleError(event json.RawMessage, taskId string, target string) error
}

type EventsHandlerImpl struct {
	ResultChannel  chan ResultEvent
	DoneChannel    chan struct{}
	ResultService  ResultService
	MetricsService MetricsFacade
}

//func (h *EventsHandlerImpl) HandleResult(event json.RawMessage, taskId string, target string) error {
//	var result ResultEvent
//	parsingErr := json.Unmarshal(event, &result)
//
//	//TODO Check behavior. When error occurs often, maybe we should send info about it to backend. But it is not critical, because we can just ignore this event and wait for next one.
//	if parsingErr != nil {
//		return parsingErr
//	}
//
//	if result.EndFlag {
//		close(h.ResultChannel)
//
//		//Synchronization of handler and event register
//		<-h.DoneChannel
//
//		h.ResultService.publishResults(target, taskId)
//	}
//
//	h.ResultChannel <- result
//
//	return nil
//}

func (h *EventsHandlerImpl) HandleResult(event json.RawMessage, taskId string, target string) error {
	var result ResultEvent
	parsingErr := json.Unmarshal(event, &result)

	//TODO Check behavior. When error occurs often, maybe we should send info about it to backend. But it is not critical, because we can just ignore this event and wait for next one.
	if parsingErr != nil {
		return parsingErr

	}
	if result.EndFlag {
		return nil
	}

	h.ResultService.publishResults(target, taskId, result)
	return nil
}

func (h *EventsHandlerImpl) HandleLog(event json.RawMessage, taskId string, target string) error {
	var logEvent LogEvent
	parsingErr := json.Unmarshal(event, &logEvent)

	if parsingErr != nil {
		return parsingErr
	}

	slog.Log(context.Background(), logEvent.Level, logEvent.Message,
		slog.String("TaskId", taskId), slog.String("Target", target), slog.String("TestId", logEvent.TestId))
	return nil
}

func (h *EventsHandlerImpl) HandleError(event json.RawMessage, taskId string, target string) error {
	return nil
}

func (h *EventsHandlerImpl) HandleMetrics(event json.RawMessage, taskId string, target string) error {
	var metricsEvent MetricsEvent
	parsingErr := json.Unmarshal(event, &metricsEvent)

	if parsingErr != nil {
		return parsingErr
	}

	//TODO remember to send labels in order defined in metrics service
	switch metricsEvent.Type {
	case Counter:
		h.MetricsService.AddCounter(metricsEvent.Name, metricsEvent.Value, metricsEvent.Labels...)
	case Histogram:
		h.MetricsService.ObserveHistogram(metricsEvent.Name, metricsEvent.Value, metricsEvent.Labels...)
	case Gauge:
		h.MetricsService.SetGauge(metricsEvent.Name, metricsEvent.Value, metricsEvent.Labels...)
	}
	return nil
}
