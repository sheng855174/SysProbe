package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	info *log.Logger
	warn *log.Logger
	err  *log.Logger
}

var Log *Logger

type LogConfig struct {
	Path       string
	MaxSizeMB  int
	MaxAge     int
	MaxBackups int
}

// log 用法
// utils.Log.Info("System Agent Service started 🚀")
// utils.Log.Warn("This is a warning ⚠️")
// utils.Log.Error("Something went wrong: %v", "timeout error")

func InitLogger(cfg LogConfig) error {
	if err := os.MkdirAll(cfg.Path, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	fileName := time.Now().Format("2006-01-02") + ".log"
	logFilePath := filepath.Join(cfg.Path, fileName)

	rotatingFile := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    cfg.MaxSizeMB,
		MaxAge:     cfg.MaxAge,
		MaxBackups: cfg.MaxBackups,
		Compress:   false,
	}

	mw := io.MultiWriter(os.Stdout, rotatingFile)

	Log = &Logger{
		info: log.New(mw, "[INFO]  ", log.LstdFlags|log.Lshortfile),
		warn: log.New(mw, "[WARN]  ", log.LstdFlags|log.Lshortfile),
		err:  log.New(mw, "[ERROR] ", log.LstdFlags|log.Lshortfile),
	}
	return nil
}

func (l *Logger) Info(msg string, v ...interface{}) {
	l.info.Output(2, fmt.Sprintf(msg, v...))
}

func (l *Logger) Warn(msg string, v ...interface{}) {
	l.warn.Output(2, fmt.Sprintf(msg, v...))
}

func (l *Logger) Error(msg string, v ...interface{}) {
	l.err.Output(2, fmt.Sprintf(msg, v...))
}
