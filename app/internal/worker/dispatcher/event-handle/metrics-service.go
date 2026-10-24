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
}

const (
	prometheusPort = "PROMETHEUS_PORT"
)

type metricType int

const (
	histogram = iota
	counter
	gauge
)

type MetricDef struct {
	Name   string
	Help   string
	Type   metricType
	Labels []string
}

var metrics = []MetricDef{
	{
		Name:   "engine_scan_duration_seconds",
		Help:   "time spent on performing one scan",
		Type:   histogram,
		Labels: []string{"test", "status"},
	},
	{
		Name:   "enigne_scan_outcomes_total",
		Help:   "measure failed and successful scans",
		Type:   counter,
		Labels: []string{"status", "reason"},
	},
	{
		Name:   "engine_vulnerabilities_found_total",
		Help:   "Total amount of found vulnerabilities grouped by threat level and test",
		Type:   counter,
		Labels: []string{"threat_level", "test"},
	},
	{
		Name:   "daemon_scans_processed_total",
		Help:   "total amount of successful scans performed by daemon",
		Type:   counter,
		Labels: nil,
	},
	{
		Name:   "daemon_internal_errors_total",
		Help:   "internal errors count, errors grouped by error code",
		Type:   counter,
		Labels: []string{"code", "component"},
	},
	{
		Name:   "daemon_active_workers",
		Help:   "amount of workers processing task at the moment",
		Type:   gauge,
		Labels: nil,
	},
	{
		Name:   "daemon_rabbit_publish_retries_total",
		Help:   "amount of retries during publishing result to broker",
		Type:   counter,
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
			Message:  "Prometheus related environment not set",
			Code:     500,
			Err:      nil,
			TaskId:   "",
			WorkerId: 0,
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
		case counter:
			service.counters[metric.Name] = promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: metric.Name,
				Help: metric.Help,
			}, metric.Labels)
		case gauge:
			service.gauges[metric.Name] = promauto.With(reg).NewGaugeVec(
				prometheus.GaugeOpts{
					Name: metric.Name,
					Help: metric.Help,
				}, metric.Labels)
		case histogram:
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
