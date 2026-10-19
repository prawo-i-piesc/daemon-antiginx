package app

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/exception"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/listener"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/listener/config"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/router"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/worker"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/worker/dispatcher"
	eventHandle "github.com/prawo-i-piesc/daemon-antiginx/app/internal/worker/dispatcher/event-handle"
)

const (
	taskChannelSizeEnvName = "TASK_CHANNEL_SIZE"
	engineCallEnvName      = "ENGINE_ANTIGINX_CALL"
	defaultWorkerPoolSize  = "10"
	engineArg              = "rawjson"
	resultsQueue           = "RABBITMQ_RESULTS_QUEUE"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	err := godotenv.Load()
	if err != nil {
		slog.Error("Cannot read .env file",
			slog.String("Error", err.Error()))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	metricsService, metricsErr := eventHandle.NewMetricsService()
	if metricsErr != nil {
		os.Exit(1)
	}
	rabbitConfig := initializeRabbitConnection()
	taskChannel, errorChannel := initializeScanChannels()
	taskRouter := initializeTaskRouter(taskChannel)
	workersWg := initializeWorkers(taskChannel, errorChannel, stop, ctx, metricsService, rabbitConfig)

	listenerErr := listener.Listen(rabbitConfig, ctx, taskRouter)
	if listenerErr != nil {
		slog.Error("Listener stopped with error", slog.Any("err", listenerErr))
	}

	slog.Info("Graceful shutdown started...")

	close(taskChannel)

	slog.Info("Waiting for workers to complete tasks...")
	workersWg.Wait()
	slog.Info("All workers completed tasks.")

	close(errorChannel)

	if rabbitConfig.ConnCh != nil {
		_ = rabbitConfig.ConnCh.Close()
		slog.Info("")
	}

	slog.Info("Application shutdown performed gracefully. GN bby girl ;)")
}

func initializeWorkers(taskChannel chan types.TaskWrapper, errorChannel chan types.ScanError, cancelFunc context.CancelFunc, ctx context.Context, metricsService *eventHandle.MetricsService, rabbitConfig *config.RabbitConfig) *sync.WaitGroup {

	engineCall := os.Getenv(engineCallEnvName)
	if engineCall == "" {
		slog.Error("ENGINE_ANTIGINX_CALL env is not set")
		os.Exit(1)
	}

	eventDispatcher := initializeDispatcher(errorChannel, 20, ctx, metricsService, rabbitConfig)
	scanListener := &worker.ScanListener{
		TaskChannel:  taskChannel,
		ErrorChannel: errorChannel,
		Orchestrator: &worker.ScanOrchestrator{
			EngineCall:      engineCall,
			EngineArg:       engineArg,
			EventDispatcher: &eventDispatcher,
		},
	}
	workersWg, workersConfigErr := worker.ConfigureWorkers(scanListener)
	if workersConfigErr != nil {
		panic(workersConfigErr)
	}

	errorListener := exception.ErrorListener{
		ErrorChannel: errorChannel,
		CancelFunc:   cancelFunc,
	}
	go errorListener.Listen()

	slog.Info("Workers configured successfully")
	return workersWg
}

func initializeTaskRouter(taskChannel chan types.TaskWrapper) router.TaskRouter {
	taskRouter := router.TaskRouter{
		ScanTaskChannel: taskChannel,
	}
	slog.Info("Router configured successfully")
	return taskRouter
}

func initializeScanChannels() (chan types.TaskWrapper, chan types.ScanError) {
	taskChannelSize := os.Getenv(taskChannelSizeEnvName)
	if taskChannelSize == "" {
		slog.Warn("TASK_CHANNEL_SIZE env is not set, fallback to default")
		taskChannelSize = defaultWorkerPoolSize
	}

	taskChannelSizeValue, taskChannelErr := strconv.Atoi(taskChannelSize)
	if taskChannelErr != nil {
		slog.Error("Error occurred during task channel size parsing",
			slog.Any("error", taskChannelErr))
		os.Exit(1)
	}
	return make(chan types.TaskWrapper, taskChannelSizeValue), make(chan types.ScanError, taskChannelSizeValue)
}

func initializeRabbitConnection() *config.RabbitConfig {

	rabbitConfig, rabbitConfigErr := config.ConfigureRabbit()
	if rabbitConfigErr != nil {
		var scanErr *types.ScanError

		if errors.As(rabbitConfigErr, &scanErr) {
			slog.Error("Error occurred during RabbitMQ configuration",
				slog.String("message", scanErr.Message),
				slog.Int("code", scanErr.Code),
				slog.Any("error", scanErr.Err),
			)
		} else {
			slog.Error("Fatal error", slog.Any("error", rabbitConfigErr))
		}
		os.Exit(1)
	}
	slog.Info("RabbitMQ configured successfully")
	return rabbitConfig
}

func initializeDispatcher(errorChannel chan types.ScanError, resultChannelSize int, ctx context.Context, metricsService *eventHandle.MetricsService, rabbitConfig *config.RabbitConfig) dispatcher.EngineEventDispatcher {
	resultChannel := make(chan eventHandle.ResultEvent, resultChannelSize)
	doneChannel := make(chan struct{})

	//eventRegister := eventHandle.EventRegister{
	//	Done: doneChannel,
	//}
	//eventRegister.StartListening(resultChannel)

	rabbitPublisher := &eventHandle.RabbitMQClient{
		Conn:        rabbitConfig.ConnCh,
		QueueName:   os.Getenv(resultsQueue),
		BackoffTime: 500,
	}

	eventsHandler := eventHandle.EventsHandlerImpl{
		ResultChannel: resultChannel,
		DoneChannel:   doneChannel,
		ResultService: &eventHandle.ResultServiceImpl{
			//Register:     eventRegister,
			Context:      ctx,
			Publisher:    rabbitPublisher,
			MaxRetries:   3,
			BackoffTime:  500,
			ErrorChannel: errorChannel,
		},
		MetricsService: metricsService,
	}
	return dispatcher.NewEngineEventDispatcher(errorChannel, &eventsHandler)
}
