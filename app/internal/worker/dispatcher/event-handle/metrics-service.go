package event_handle

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsFacade interface {
	AddCounter(metricName string, value float64, labels ...string)
	ObserveHistogram(metricName string, value float64, labels ...string)
	SetGauge(metricName string, value float64, labels ...string)
	IncGauge(metricName string, labels ...string)
	DecGauge(metricName string, labels ...string)
}

const (
	prometheusPort = "PROMETHEUS_PORT"
)

type MetricType int

const (
	Histogram MetricType = iota
	Counter
	Gauge
)

const (
	InternalErrorsTotal = "daemon_internal_errors_total"
	ActiveWorkers       = "daemon_active_workers"
	RabbitResultRetry   = "daemon_rabbit_publish_retries_total"
)

type MetricDef struct {
	Name   string
	Help   string
	Type   MetricType
	Labels []string
}

var metrics = []MetricDef{
	{
		Name:   "engine_scan_duration_seconds",
		Help:   "time spent on performing one scan",
		Type:   Histogram,
		Labels: []string{},
	},
	{
		Name:   "engine_test_duration_seconds",
		Help:   "time spent on performing one test",
		Type:   Histogram,
		Labels: []string{"test"},
	},
	{
		Name:   "engine_scan_outcomes_total",
		Help:   "measure failed and successful scans",
		Type:   Counter,
		Labels: []string{"status"},
	},
	{
		Name:   "engine_vulnerabilities_found_total",
		Help:   "Total amount of found vulnerabilities grouped by threat level and test",
		Type:   Counter,
		Labels: []string{"threat_level", "test"},
	},
	{
		Name:   "daemon_internal_errors_total",
		Help:   "internal errors count, errors grouped by error code",
		Type:   Counter,
		Labels: []string{"code", "component"},
	},
	{
		Name:   "daemon_active_workers",
		Help:   "amount of workers processing task at the moment",
		Type:   Gauge,
		Labels: nil,
	},
	{
		Name:   "daemon_rabbit_publish_retries_total",
		Help:   "amount of retries during publishing result to broker",
		Type:   Counter,
		Labels: nil,
	},
}

type MetricsService struct {
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

func NewMetricsService() (*MetricsService, error) {
	prometheusPort := os.Getenv(prometheusPort)
	if prometheusPort == "" {
		slog.Error("PROMETHEUS_PORT env is not set")
		return nil, &types.ScanError{
			Message:   "Prometheus related environment not set",
			Code:      500,
			Err:       nil,
			TaskId:    "",
			WorkerId:  0,
			Component: "metrics-service",
		}
	}

	service := MetricsService{
		counters:   make(map[string]*prometheus.CounterVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	for _, metric := range metrics {
		switch metric.Type {
		case Counter:
			service.counters[metric.Name] = promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: metric.Name,
				Help: metric.Help,
			}, metric.Labels)
		case Gauge:
			service.gauges[metric.Name] = promauto.With(reg).NewGaugeVec(
				prometheus.GaugeOpts{
					Name: metric.Name,
					Help: metric.Help,
				}, metric.Labels)
		case Histogram:
			service.histograms[metric.Name] = promauto.With(reg).NewHistogramVec(
				prometheus.HistogramOpts{
					Name: metric.Name,
					Help: metric.Help,
				}, metric.Labels)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	go func() {
		if err := http.ListenAndServe(prometheusPort, mux); err != nil {
			slog.Error("Prometheus server failed", slog.Any("Error", err))
		}
	}()
	return &service, nil
}

func (m *MetricsService) AddCounter(metricName string, value float64, labels ...string) {
	if counter, exists := m.counters[metricName]; exists {
		counter.WithLabelValues(labels...).Add(value)
	} else {
		slog.Warn("Counter not found", slog.String("name", metricName))
	}
}

func (m *MetricsService) ObserveHistogram(metricName string, value float64, labels ...string) {
	if hist, exists := m.histograms[metricName]; exists {
		hist.WithLabelValues(labels...).Observe(value)
	} else {
		slog.Warn("Histogram not found", slog.String("name", metricName))
	}
}

func (m *MetricsService) SetGauge(metricName string, value float64, labels ...string) {
	if gauge, exists := m.gauges[metricName]; exists {
		gauge.WithLabelValues(labels...).Set(value)
	} else {
		slog.Warn("Gauge not found", slog.String("name", metricName))
	}
}

func (m *MetricsService) IncGauge(metricName string, labels ...string) {
	if gauge, exists := m.gauges[metricName]; exists {
		gauge.WithLabelValues(labels...).Inc()
	} else {
		slog.Warn("Gauge not found", slog.String("name", metricName))
	}
}

func (m *MetricsService) DecGauge(metricName string, labels ...string) {
	if gauge, exists := m.gauges[metricName]; exists {
		gauge.WithLabelValues(labels...).Dec()
	} else {
		slog.Warn("Gauge not found", slog.String("name", metricName))
	}
}
