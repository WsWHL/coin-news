package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"time"
)

var log = logrus.New()

type Hook struct {
	Writer    io.Writer
	Formatter logrus.Formatter
	Level     []logrus.Level
}

func (h *Hook) Fire(entry *logrus.Entry) error {
	line, err := h.Formatter.Format(entry)
	if err != nil {
		return err
	}
	if _, err = h.Writer.Write(line); err != nil {
		return err
	}

	return nil
}

func (h *Hook) Levels() []logrus.Level {
	return h.Level
}

func newHook(writer io.Writer, formatter logrus.Formatter, level logrus.Level) *Hook {
	var levels []logrus.Level
	for _, l := range logrus.AllLevels {
		if l <= level {
			levels = append(levels, l)
		}
	}
	return &Hook{
		Writer:    writer,
		Formatter: formatter,
		Level:     levels,
	}
}

func init() {
	log.SetLevel(logrus.DebugLevel)
	log.SetOutput(io.Discard)

	fileName := fmt.Sprintf("./logs/news_%s.log", time.Now().Format("20060102"))
	logFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %s", fileName)
		panic(err)
	}

	fileName = fmt.Sprintf("./logs/news-err_%s.log", time.Now().Format("20060102"))
	errLogFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %s", fileName)
		panic(err)
	}

	log.AddHook(newHook(
		logFile,
		&logrus.JSONFormatter{},
		logrus.DebugLevel,
	))

	log.AddHook(newHook(
		errLogFile,
		&logrus.JSONFormatter{},
		logrus.ErrorLevel,
	))

	log.AddHook(newHook(
		os.Stderr,
		&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		},
		logrus.DebugLevel,
	))
}

func GetLogger() *logrus.Logger {
	return log
}

func Debug(msg string) {
	log.Debug(msg)
}

func Debugf(f string, args ...interface{}) {
	log.Debugf(f, args...)
}

func Info(msg string) {
	log.Info(msg)
}

func Infof(f string, args ...interface{}) {
	log.Infof(f, args...)
}

func Warnf(f string, args ...interface{}) {
	log.Warnf(f, args...)
}

func Warn(msg string) {
	log.Warn(msg)
}

func Errorf(f string, args ...interface{}) {
	log.Errorf(f, args...)
}

func Error(msg string) {
	log.Error(msg)
}
