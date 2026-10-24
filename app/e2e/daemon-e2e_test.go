package e2e

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/prawo-i-piesc/daemon-antiginx/app"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	eventHandle "github.com/prawo-i-piesc/daemon-antiginx/app/internal/worker/dispatcher/event-handle"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

const (
	tasksQueue    = "tasks_queue"
	resultsQueue  = "results_queue"
	mockBehaviour = "TEST_BEHAVIOUR"
)

//go:embed invalid-task-payload.json
var invalidTaskPayload string

var task = types.ScanTask{
	TaskId: "test-happy-123",
	Target: "example.com",
	Parameters: []*types.CommandParameter{
		{
			Name:      "test_param",
			Arguments: []string{"-v", "-p", "80"},
		},
	},
}

type E2ETestSuite struct {
	suite.Suite
	amqpURL         string
	enginePath      string
	rabbitContainer *rabbitmq.RabbitMQContainer
	ctxTC           context.Context
	amqpConn        *amqp.Connection
	amqpChannel     *amqp.Channel
}

func (s *E2ETestSuite) SetupSuite() {
	container, amqpURL, ctxTC := setupRabbit(s.T())

	enginePath := buildMockEngine(s.T())

	s.enginePath = enginePath
	s.ctxTC = ctxTC
	s.amqpURL = amqpURL
	s.rabbitContainer = container
}

func (s *E2ETestSuite) TearDownSuite() {
	err := s.rabbitContainer.Terminate(s.ctxTC)
	require.NoError(s.T(), err)
}

func (s *E2ETestSuite) SetupTest() {
	conn, err := amqp.Dial(s.amqpURL)
	s.Require().NoError(err)
	s.amqpConn = conn

	ch, err := conn.Channel()
	s.Require().NoError(err)
	s.amqpChannel = ch

	args := amqp.Table{"x-queue-type": "quorum", "x-delivery-limit": 1}
	_, err = ch.QueueDeclare(tasksQueue, true, false, false, false, args)
	s.Require().NoError(err)
	_, err = ch.QueueDeclare(resultsQueue, true, false, false, false, nil)
	s.Require().NoError(err)

	_, err = ch.QueuePurge(tasksQueue, false)
	s.Require().NoError(err)
	_, err = ch.QueuePurge(resultsQueue, false)
	s.Require().NoError(err)
}

func (s *E2ETestSuite) TearDownTest() {
	if s.amqpChannel != nil {
		_ = s.amqpChannel.Close()
	}
	if s.amqpConn != nil {
		_ = s.amqpConn.Close()
	}
}

func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

func (s *E2ETestSuite) TestE2E_HappyPath() {
	setupEnvironment(s.T(), s.enginePath, s.amqpURL, "SUCCESS")
	ctx, cancelDaemon := context.WithCancel(context.Background())
	defer cancelDaemon()

	daemonErrCh := make(chan error, 1)
	go func() {
		daemonErrCh <- app.Run(ctx)
	}()

	msgs, err := s.amqpChannel.Consume(resultsQueue, "", true, false, false, false, nil)
	require.NoError(s.T(), err)

	body, err := json.Marshal(task)
	require.NoError(s.T(), err)

	err = s.amqpChannel.PublishWithContext(s.ctxTC, "", tasksQueue, false, false, amqp.Publishing{
		ContentType: "application/json",
		MessageId:   task.TaskId,
		Body:        body,
	})
	require.NoError(s.T(), err)

	select {
	case msg := <-msgs:
		var result eventHandle.ScanContext
		err := json.Unmarshal(msg.Body, &result)
		require.NoError(s.T(), err, "Unserializable event sent by daemon:\n%s", err)
	case <-time.After(5 * time.Second):
		s.T().Fatal("Timeout")
	}

	cancelDaemon()
	err = <-daemonErrCh
	require.NoError(s.T(), err)
}

func (s *E2ETestSuite) TestE2E_EngineCrash() {
	setupEnvironment(s.T(), s.enginePath, s.amqpURL, "FAILURE")
	ctx, cancelDaemon := context.WithCancel(context.Background())
	defer cancelDaemon()

	daemonErrCh := make(chan error, 1)
	go func() {
		daemonErrCh <- app.Run(ctx)
	}()

	msgs, err := s.amqpChannel.Consume(resultsQueue, "", true, false, false, false, nil)
	require.NoError(s.T(), err)

	body, err := json.Marshal(task)
	require.NoError(s.T(), err)

	err = s.amqpChannel.PublishWithContext(s.ctxTC, "", tasksQueue, false, false, amqp.Publishing{
		ContentType: "application/json",
		MessageId:   task.TaskId,
		Body:        body,
	})
	require.NoError(s.T(), err)

	select {
	case msg := <-msgs:
		s.T().Fatalf("Expected no messages on results_queue, but received: %s", string(msg.Body))
	case <-time.After(2 * time.Second):
	}

	cancelDaemon()
	err = <-daemonErrCh
	require.NoError(s.T(), err)
}

func (s *E2ETestSuite) TestE2E_InvalidTaskPayload() {
	setupEnvironment(s.T(), s.enginePath, s.amqpURL, "SUCCESS")

	var logBuffer bytes.Buffer
	multiWriter := io.MultiWriter(os.Stdout, &logBuffer)
	logger := slog.New(slog.NewTextHandler(multiWriter, nil))
	oldLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldLogger)

	ctx, cancelDaemon := context.WithCancel(context.Background())
	defer cancelDaemon()

	daemonErrCh := make(chan error, 1)
	go func() {
		daemonErrCh <- app.Run(ctx)
	}()

	msgs, err := s.amqpChannel.Consume(resultsQueue, "", true, false, false, false, nil)
	require.NoError(s.T(), err)

	err = s.amqpChannel.PublishWithContext(s.ctxTC, "", tasksQueue, false, false, amqp.Publishing{
		ContentType: "application/json",
		MessageId:   task.TaskId,
		Body:        []byte(invalidTaskPayload),
	})
	require.NoError(s.T(), err)

	select {
	case msg := <-msgs:
		s.T().Fatalf("Expected no messages on results_queue, but received: %s", string(msg.Body))
	case <-time.After(2 * time.Second):
	}

	cancelDaemon()
	err = <-daemonErrCh
	s.Require().NoError(err)

	s.Require().Contains(logBuffer.String(), "Scan task parsing error")

	qInfo, err := s.amqpChannel.QueueDeclarePassive(tasksQueue, true, false, false, false, nil)
	s.Require().NoError(err)
	s.Require().Equal(0, qInfo.Messages, "Queue tasks_queue should be empty, but message was requeued")
}

func (s *E2ETestSuite) TestE2E_InvalidEngineEventPayload() {
	setupEnvironment(s.T(), s.enginePath, s.amqpURL, "INVALID_EVENT")

	var logBuffer bytes.Buffer
	multiWriter := io.MultiWriter(os.Stdout, &logBuffer)
	logger := slog.New(slog.NewTextHandler(multiWriter, nil))
	oldLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldLogger)

	ctx, cancelDaemon := context.WithCancel(context.Background())
	defer cancelDaemon()

	body, err := json.Marshal(task)
	require.NoError(s.T(), err)

	daemonErrCh := make(chan error, 1)
	go func() {
		daemonErrCh <- app.Run(ctx)
	}()

	msgs, err := s.amqpChannel.Consume(resultsQueue, "", true, false, false, false, nil)
	require.NoError(s.T(), err)

	err = s.amqpChannel.PublishWithContext(s.ctxTC, "", tasksQueue, false, false, amqp.Publishing{
		ContentType: "application/json",
		MessageId:   task.TaskId,
		Body:        body,
	})
	require.NoError(s.T(), err)

	select {
	case msg := <-msgs:
		s.T().Fatalf("Expected no messages on results_queue, but received: %s", string(msg.Body))
	case <-time.After(2 * time.Second):
	}

	s.Require().Contains(logBuffer.String(), "Faulty engine event, cannot parse (skipped)")

	qInfo, err := s.amqpChannel.QueueDeclarePassive(tasksQueue, true, false, false, false, nil)
	s.Require().NoError(err)
	s.Require().Equal(0, qInfo.Messages, "Queue tasks_queue should be empty, but message was requeued")
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

func setupRabbit(t *testing.T) (*rabbitmq.RabbitMQContainer, string, context.Context) {
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

	amqpURL, err := rabbitContainer.AmqpURL(ctxTC)
	require.NoError(t, err)

	return rabbitContainer, amqpURL, ctxTC
}
func setupEnvironment(t *testing.T, enginePath, amqpURL string, engineBehaviour string) {
	t.Helper()

	t.Setenv("TASK_CHANNEL_SIZE", "5")
	t.Setenv("ENGINE_ANTIGINX_CALL", enginePath)
	t.Setenv("WORKER_POOL", "2")
	t.Setenv("RABBITMQ_URL", amqpURL)
	t.Setenv("RABBITMQ_QUEUE_NAME", tasksQueue)
	t.Setenv("RABBITMQ_RESULTS_QUEUE", resultsQueue)
	t.Setenv("PREFETCH_COUNT", "2")
	t.Setenv("PROMETHEUS_PORT", ":9091")

	t.Setenv(mockBehaviour, engineBehaviour)
}
