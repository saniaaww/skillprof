package config

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger() zerolog.Logger {
	logFile := &lumberjack.Logger{
		Filename:   "logs/app.log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	logger := zerolog.New(multiWriter).
		With().
		Timestamp().
		Caller().
		Logger()

	return logger
}
