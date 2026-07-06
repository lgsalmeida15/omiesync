package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func New(appEnv string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	level := zerolog.InfoLevel
	if lvl, err := zerolog.ParseLevel(os.Getenv("LOG_LEVEL")); err == nil {
		level = lvl
	}
	zerolog.SetGlobalLevel(level)

	if appEnv != "production" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			With().
			Timestamp().
			Caller().
			Logger()
	}

	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger()
}
