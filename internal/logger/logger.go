package logger

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

func Setup() {
	fileWriter := &lumberjack.Logger{
		Filename: "logs/app.log",
		MaxSize:  10,
		MaxAge:   30,
		Compress: true,
	}

	multiWriter := io.MultiWriter(os.Stdout, fileWriter)

	slog.SetDefault(slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
}
