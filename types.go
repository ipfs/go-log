package log

import (
	logrus "github.com/sirupsen/logrus"
)

var DefaultJSONFormat logrus.Formatter

var DefaultTextFormat logrus.Formatter

func init() {
	DefaultJSONFormat = &logrus.JSONFormatter{
		DisableTimestamp: false,
	}

	DefaultTextFormat = &logrus.TextFormatter{
		FullTimestamp: true,
	}
}

type Entry = logrus.Entry

type Fields = logrus.Fields

type JSONFormat = logrus.JSONFormatter

type TextFormat = logrus.TextFormatter

type Formatter = logrus.Formatter

type Level = logrus.Level
