package log

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/luckycat0426/bililive-go/src/instance"
	"github.com/luckycat0426/bililive-go/src/interfaces"
)

func New(ctx context.Context) *interfaces.Logger {
	inst := instance.GetInstance(ctx)
	logLevel := logrus.InfoLevel
	if inst.Config.Debug {
		logLevel = logrus.DebugLevel
	}
	logger := &interfaces.Logger{Logger: &logrus.Logger{
		Out: os.Stderr,
		Formatter: &logrus.TextFormatter{
			DisableColors:   true,
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		},
		Hooks: make(logrus.LevelHooks),
		Level: logLevel,
	}}

	inst.Logger = logger

	return logger
}
