package app

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/prawo-i-piesc/scan-daemon/app/internal/common/types"
	"github.com/prawo-i-piesc/scan-daemon/app/internal/listener"
	"github.com/prawo-i-piesc/scan-daemon/app/internal/listener/config"
	"github.com/prawo-i-piesc/scan-daemon/app/internal/router"
	"github.com/prawo-i-piesc/scan-daemon/app/internal/worker"

	"github.com/joho/godotenv"
)

const (
	taskChannelSizeEnvName = "TASK_CHANNEL_SIZE"
	defaultWorkerPoolSize  = "10"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	err := godotenv.Load()
	if err != nil {
		slog.Error("Cannot read .env file", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rabbitConfig, rabbitConfigErr := config.ConfigureRabbit()
	if rabbitConfigErr != nil {
		var scanErr *types.ScanError

		if errors.As(err, &scanErr) {
			slog.Error("Error occurred during RabbitMQ configuration",
				slog.String("message", scanErr.Message),
				slog.Int("code", scanErr.Code),
				slog.Any("error", scanErr.Err),
			)
		} else {
			slog.Error("Fatal error", slog.Any("error", err))
		}
		os.Exit(1)
	}
	slog.Info("RabbitMQ configured successfully")

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

	taskChannel := make(chan types.TaskWrapper, taskChannelSizeValue)

	taskRouter := router.TaskRouter{
		ScanTaskChannel: taskChannel,
	}
	slog.Info("Router configured successfully")

	scanListener := worker.ScanListener{
		TaskChannel: taskChannel,
	}
	workersConfigErr := worker.ConfigureWorkers(scanListener)
	if workersConfigErr != nil {
		os.Exit(1)
	}
	slog.Info("Workers configured successfully")

	listenerErr := listener.Listen(rabbitConfig, ctx, taskRouter)
	if listenerErr != nil {
		os.Exit(1)
	}

}
