package event_handle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsService_EnvNotSet(t *testing.T) {
	t.Setenv("PROMETHEUS_PORT", "")
	svc, err := NewMetricsService()

	require.Error(t, err)
	require.Nil(t, svc)
	assert.Equal(t, "Prometheus related environment not set", err.Error())
}

func TestNewMetricsService_SuccessAndMethods(t *testing.T) {
	t.Setenv("PROMETHEUS_PORT", ":0")
	svc, err := NewMetricsService()

	require.NoError(t, err)
	require.NotNil(t, svc)

	assert.NotPanics(t, func() {
		svc.AddCounter(InternalErrorsTotal, 1, "500", "test")
		svc.ObserveHistogram("engine_scan_duration_seconds", 1.5)
		svc.SetGauge(ActiveWorkers, 5)
		svc.IncGauge(ActiveWorkers)
		svc.DecGauge(ActiveWorkers)
		
		svc.AddCounter("non_existing_counter", 1)
		svc.ObserveHistogram("non_existing_hist", 1.5)
		svc.SetGauge("non_existing_gauge", 5)
		svc.IncGauge("non_existing_gauge")
		svc.DecGauge("non_existing_gauge")
	})
}
