package logger

import (
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/router-architects/network-topology/internal/config"
)

func Init(cfg config.Config) {
	lvl, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	logrus.SetLevel(lvl)
	logrus.SetOutput(os.Stdout)
	logrus.SetReportCaller(true)

	if cfg.LogJSON {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	if cfg.LogFilePath != "" {
		logrus.AddHook(&rotateHook{
			logger: &lumberjack.Logger{
				Filename:   cfg.LogFilePath,
				MaxSize:    50, // MB
				MaxBackups: 5,
				MaxAge:     30, // days
				Compress:   true,
			},
		})
	}
}

type rotateHook struct {
	logger *lumberjack.Logger
}

func (h *rotateHook) Levels() []logrus.Level { return logrus.AllLevels }
func (h *rotateHook) Fire(e *logrus.Entry) error {
	// write a copy to file sink
	b, err := e.Logger.Formatter.Format(e)
	if err != nil {
		return err
	}
	_, err = h.logger.Write(b)
	return err
}
