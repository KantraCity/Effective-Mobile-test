package logger

import (
	"github.com/rs/zerolog"
	"os"
)

func New(level string) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	return zerolog.New(os.Stdout).Level(lvl).With().Timestamp().Logger()
}
