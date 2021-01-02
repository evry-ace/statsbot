package main

import (
	"github.com/sirupsen/logrus"
)

func setupLogging() {
	c, _ := ConfigFromEnvironment()

	switch c.LogLevel {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	default:
		logrus.WithField("log-level", c.LogLevel).Warning("invalid log level. defaulting to info.")
		logrus.SetLevel(logrus.InfoLevel)
	}

	switch c.LogFormat {
	case "text":
		logrus.SetFormatter(new(logrus.TextFormatter))
	case "json":
		logrus.SetFormatter(new(logrus.JSONFormatter))
	default:
		logrus.WithField("log-format", c.LogFormat).Warning("invalid log format. defaulting to text.")
		logrus.SetFormatter(new(logrus.TextFormatter))
	}
}
