package main // ZMIANA 1: Zmiana z 'app' na 'main'

import (
	"errors"
	"log/slog"
	"os"

	"github.com/prawo-i-piesc/scan-daemon/app/internal/common"
	"github.com/prawo-i-piesc/scan-daemon/app/internal/listener/config"

	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	err := godotenv.Load()
	if err != nil {
		slog.Error("Cannot read .env file", "error", err)
		os.Exit(1)
	}

	//configuration ignored to silence compiler
	_, err = config.ConfigureRabbit()
	if err != nil {
		var scanErr *common.ScanError

		if errors.As(err, &scanErr) {
			slog.Error("Error occurred during RabbitMQ configuration",
				slog.String("message", scanErr.Message),
				slog.Int("code", scanErr.Code),
			)
		} else {
			slog.Error("Fatal error", slog.Any("error", err))
		}
		os.Exit(1)
	}
	slog.Info("RabbitMQ configured successfully")

}
