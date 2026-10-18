package config

import (
	"testing"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	"github.com/stretchr/testify/require"
)

const (
	envURL      = "amqp://guest:guest@localhost:5672/"
	prefetchEnv = "10"
	queue       = "test-queue"
)

var tests = []struct {
	name           string
	envURL         string
	envPrefetch    string
	envQueue       string
	expectedErrMsg string
}{
	{
		name:           "Missing RABBITMQ_URL",
		envURL:         "",
		expectedErrMsg: "RABBITMQ_URL env is not set",
	},
	{
		name:           "Invalid PREFETCH_COUNT",
		envURL:         envURL,
		envPrefetch:    "not-a-number",
		expectedErrMsg: "PREFETCH_COUNT has to be a number not-a-number",
	},
	{
		name:           "Missing RABBITMQ_QUEUE_NAME",
		envURL:         envURL,
		envPrefetch:    prefetchEnv,
		envQueue:       "",
		expectedErrMsg: "RABBITMQ_QUEUE_NAME env is not set",
	},
	{
		name:           "Dial error on invalid URL",
		envURL:         "amqp://invalid-host:9999/",
		envPrefetch:    prefetchEnv,
		envQueue:       queue,
		expectedErrMsg: "Cannot create TCP dial connection with amqp://invalid-host:9999/",
	},
}

func TestConfigureRabbit_EnvValidation(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("RABBITMQ_URL", tt.envURL)
			t.Setenv("RABBITMQ_QUEUE_NAME", tt.envQueue)
			if tt.envPrefetch != "" {
				t.Setenv("PREFETCH_COUNT", tt.envPrefetch)
			}

			cfg, err := ConfigureRabbit()

			require.Error(t, err)
			require.Nil(t, cfg)

			var scanErr *types.ScanError
			require.ErrorAs(t, err, &scanErr)
			require.Contains(t, scanErr.Message, tt.expectedErrMsg)
		})
	}
}
