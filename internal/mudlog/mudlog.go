package mudlog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/natefinch/lumberjack"
)

var (
	logLevel = new(slog.LevelVar) // goroutine safe way to change log levels

	// slogInstance logs to stderr until SetupLogger replaces it, so a caller
	// (a test, a tool) that never sets up logging can't hit a nil logger. It is
	// atomic so a goroutine already logging never races SetupLogger.
	slogInstance atomic.Pointer[slog.Logger]
)

func init() {
	slogInstance.Store(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))
}

type teeLogger interface {
	Println(level string, v ...any)
}

func SetLogLevel(lvl string) {

	if len(lvl) > 0 {
		if lvl[0:1] == `M` {
			logLevel.Set(slog.LevelInfo)
			return
		} else if lvl[0:1] == `L` {
			logLevel.Set(slog.LevelWarn)
			return
		}
	}

	logLevel.Set(slog.LevelDebug)

}

func SetupLogger(inGameLogger teeLogger, logLevel string, logPath string, colorLogs bool) {

	SetLogLevel(strings.ToUpper(logLevel))

	// No filepath? Write to Stderr.
	if logPath == `` {
		slogInstance.Store(slog.New(
			getLogHandler(os.Stderr, inGameLogger, colorLogs),
		))
		return
	}

	// Setup file logging
	fileInfo, err := os.Stat(logPath)
	if err == nil {
		if fileInfo.IsDir() {
			panic(fmt.Errorf("log file path is a directory: %s", logPath))
		}

	} else if os.IsNotExist(err) {
		// File does not exist; check if the directory exists
		dir := filepath.Dir(logPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			panic(fmt.Errorf("directory for log file does not exist: %s", dir))
		}
	} else {
		// Some other error
		panic(fmt.Errorf("error accessing log file path: %v", err))
	}

	lj := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100,  // Maximum size in megabytes before rotation
		MaxBackups: 10,   // Maximum number of old log files to retain
		Compress:   true, // Compress rotated files
	}

	slogInstance.Store(slog.New(
		getLogHandler(lj, inGameLogger, colorLogs),
	))

}

func Debug(msg string, args ...any) {
	slogInstance.Load().Log(context.Background(), slog.LevelDebug, msg, args...)
}

func Info(msg string, args ...any) {
	slogInstance.Load().Log(context.Background(), slog.LevelInfo, msg, args...)
}

func Warn(msg string, args ...any) {
	slogInstance.Load().Log(context.Background(), slog.LevelWarn, msg, args...)
}

func Error(msg string, args ...any) {
	slogInstance.Load().Log(context.Background(), slog.LevelError, msg, args...)
}
