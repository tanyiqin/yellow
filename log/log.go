package log

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

var baseLogger zerolog.Logger

func init() {
	now := time.Now()
	fileName := fmt.Sprintf("log/%v_%v_%v_%v", now.Year(), now.Month(), now.Day(), now.Hour())
	logFile, err := os.Create(fileName)
	if err != nil {
		fmt.Println("logger init err =", err)
		os.Exit(1)
	}
	baseLogger = log.Output(zerolog.MultiLevelWriter(logFile, os.Stdout))
}

func Panic(format string, a ...interface{}) {
	baseLogger.Panic().Msgf(format, a...)
}

func Error(format string, a ...interface{}) {
	baseLogger.Error().Msgf(format, a...)
}

func Info(format string, a ...interface{}) {
	baseLogger.Info().Msgf(format, a...)
}

func Debug(format string, a ...interface{}) {
	baseLogger.Debug().Msgf(format, a...)
}