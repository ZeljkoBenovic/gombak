package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/ZeljkoBenovic/gombak/pkg/config"
)

var logLevels = map[string]slog.Level{
	"info":  slog.LevelInfo,
	"debug": slog.LevelDebug,
	"error": slog.LevelError,
	"warn":  slog.LevelWarn,
}

type Logger struct {
	*slog.Logger

	conf config.Config
}

func New(conf config.Config) (*Logger, error) {
	var (
		l       *slog.Logger
		logFile io.Writer
		err     error
	)

	logFile, err = logWriter(conf.Logger.File)
	if err != nil {
		return nil, err
	}

	switch conf.Logger.JSONOutput {
	case true:
		l = slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{
			Level: logLevels[strings.ToLower(conf.Logger.Level)],
		}))
	default:
		l = slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
			Level: logLevels[strings.ToLower(conf.Logger.Level)],
		}))
	}

	return &Logger{
		Logger: l,
		conf:   conf,
	}, nil
}

func (l *Logger) Name(name string) *Logger {
	l.Logger = l.Logger.With(slog.String("module", name))

	return l
}

func logWriter(logFileName string) (io.Writer, error) {
	switch logFileName {
	case "":
		return os.Stdout, nil
	default:
		file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return nil, fmt.Errorf("could not open specified file: %w", err)
		}

		return file, nil
	}
}
