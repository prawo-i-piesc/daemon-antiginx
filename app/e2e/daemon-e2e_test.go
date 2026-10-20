//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/prawo-i-piesc/daemon-antiginx/app"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	event_handle "github.com/prawo-i-piesc/daemon-antiginx/app/internal/worker/dispatcher/event-handle"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

const (
	tasksQueue    = "tasks_queue"
	resultsQueue  = "results_queue"
	mockBehaviour = "TEST_BEHAVIOUR"
)

func TestE2E_HappyPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		if os.Getenv("DOCKER_HOST") == "" {
			err := os.Setenv("DOCKER_HOST", "npipe:////./pipe/docker_engine")
			require.NoError(t, err)
		}
		err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
		require.NoError(t, err)
	}
	ctxTC := context.Background()
	rabbitContainer, err := rabbitmq.Run(ctxTC, "rabbitmq:3-management-alpine")
	require.NoError(t, err)
	defer rabbitContainer.Terminate(ctxTC)

	amqpURL, err := rabbitContainer.AmqpURL(ctxTC)
	require.NoError(t, err)

	enginePath := buildMockEngine(t)
	setupEnvironment(t, enginePath, amqpURL)

	ctx, cancelDaemon := context.WithCancel(context.Background())
	defer cancelDaemon()

	daemonErrCh := make(chan error, 1)
	go func() {
		daemonErrCh <- app.Run(ctx)
	}()

	conn, err := amqp.Dial(amqpURL)
	require.NoError(t, err)
	defer conn.Close()

	ch, err := conn.Channel()
	require.NoError(t, err)
	defer ch.Close()

	_, err = ch.QueueDeclare(tasksQueue, true, false, false, false, nil)
	require.NoError(t, err)
	_, err = ch.QueueDeclare(resultsQueue, true, false, false, false, nil)
	require.NoError(t, err)

	msgs, err := ch.Consume(resultsQueue, "", true, false, false, false, nil)
	require.NoError(t, err)

	task := types.ScanTask{
		TaskId: "test-happy-123",
		Target: "example.com",
		Parameters: []*types.CommandParameter{
			{
				Name:      "test_param",
				Arguments: []string{"-v", "-p", "80"},
			},
		},
	}
	body, err := json.Marshal(task)
	require.NoError(t, err)

	err = ch.PublishWithContext(ctxTC, "", tasksQueue, false, false, amqp.Publishing{
		ContentType: "application/json",
		MessageId:   task.TaskId,
		Body:        body,
	})
	require.NoError(t, err)

	select {
	case msg := <-msgs:
		var result event_handle.ScanContext
		err := json.Unmarshal(msg.Body, &result)
		require.NoError(t, err, "Unserializable event sent by daemon:\n%s", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout")
	}

	cancelDaemon()
	err = <-daemonErrCh
	require.NoError(t, err)
}

func buildMockEngine(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	binaryName := "mock-engine"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	enginePath := filepath.Join(tempDir, binaryName)
	sourcePath := filepath.Join("..", "testdata", "mock-engine.go")

	cmd := exec.Command("go", "build", "-o", enginePath, sourcePath)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	return enginePath
}

func setupEnvironment(t *testing.T, enginePath, amqpURL string) {
	t.Helper()

	t.Setenv("TASK_CHANNEL_SIZE", "5")
	t.Setenv("ENGINE_ANTIGINX_CALL", enginePath)
	t.Setenv("WORKER_POOL", "2")
	t.Setenv("RABBITMQ_URL", amqpURL)
	t.Setenv("RABBITMQ_QUEUE_NAME", tasksQueue)
	t.Setenv("RABBITMQ_RESULTS_QUEUE", resultsQueue)
	t.Setenv("PREFETCH_COUNT", "2")
	t.Setenv("PROMETHEUS_PORT", ":9091")

	t.Setenv(mockBehaviour, "SUCCESS")
}
