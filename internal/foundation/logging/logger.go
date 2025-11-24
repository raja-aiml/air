package logging

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger initializes the global zerolog logger with the specified level
func InitLogger(levelStr string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	level := parseLevel(levelStr)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).Level(level)
}

func parseLevel(levelStr string) zerolog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
