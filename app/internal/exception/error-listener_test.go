package exception

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DummyMetrics struct{}

func (m *DummyMetrics) AddCounter(n string, v float64, l ...string)       {}
func (m *DummyMetrics) ObserveHistogram(n string, v float64, l ...string) {}
func (m *DummyMetrics) SetGauge(n string, v float64, l ...string)         {}
func (m *DummyMetrics) IncGauge(n string, l ...string)                    {}
func (m *DummyMetrics) DecGauge(n string, l ...string)                    {}

var tests = []struct {
	name           string
	errCode        int
	errMsg         string
	expectedLogMsg string
	expectCancel   bool
}{
	{
		name:           "Should log warning and not cancel on code 400",
		errCode:        400,
		errMsg:         "Domain error occurred",
		expectedLogMsg: "Domain error occurred",
		expectCancel:   false,
	},
	{
		name:           "Should log error and cancel on code 500",
		errCode:        500,
		errMsg:         "Connection dropped",
		expectedLogMsg: "Fatal error, application shutdown",
		expectCancel:   true,
	},
	{
		name:           "Should completely ignore unknown error codes",
		errCode:        999,
		errMsg:         "Unknown weird error",
		expectedLogMsg: "",
		expectCancel:   false,
	},
}

func TestErrorListener_Listen(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			multiWriter := io.MultiWriter(os.Stdout, &logBuffer)
			logger := slog.New(slog.NewTextHandler(multiWriter, nil))
			oldLogger := slog.Default()
			slog.SetDefault(logger)
			defer slog.SetDefault(oldLogger)

			cancelCalled := false
			cancelFunc := func() {
				cancelCalled = true
			}

			errChan := make(chan types.ScanError, 1)

			listener := &ErrorListener{
				ErrorChannel:   errChan,
				CancelFunc:     cancelFunc,
				MetricsService: &DummyMetrics{},
			}

			errChan <- types.ScanError{
				Code:     tt.errCode,
				Message:  tt.errMsg,
				Err:      errors.New("internal cause"),
				TaskId:   "task-test",
				WorkerId: 1,
			}
			close(errChan)

			listener.Listen()

			require.Equal(t, tt.expectCancel, cancelCalled)

			if tt.expectedLogMsg != "" {
				assert.Contains(t, logBuffer.String(), tt.expectedLogMsg)
			} else {
				assert.NotContains(t, logBuffer.String(), tt.errMsg)
			}
		})
	}
}

func TestErrorListener_Listen_MultipleErrors(t *testing.T) {
	cancelCount := 0
	cancelFunc := func() {
		cancelCount++
	}

	errChan := make(chan types.ScanError, 3)

	listener := &ErrorListener{
		ErrorChannel:   errChan,
		CancelFunc:     cancelFunc,
		MetricsService: &DummyMetrics{},
	}

	errChan <- types.ScanError{
		Message:  "Warn 1",
		Code:     400,
		Err:      errors.New("test error"),
		TaskId:   "1",
		WorkerId: 0,
	}
	errChan <- types.ScanError{
		Message:  "Warn 2",
		Code:     400,
		Err:      errors.New("test error"),
		TaskId:   "2",
		WorkerId: 0,
	}
	errChan <- types.ScanError{
		Message:  "Fatal",
		Code:     500,
		Err:      errors.New("test error"),
		TaskId:   "3",
		WorkerId: 0,
	}
	close(errChan)

	listener.Listen()

	require.Equal(t, 1, cancelCount)
}
