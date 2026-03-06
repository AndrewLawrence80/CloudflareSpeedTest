package log

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/common"
)

const logFileMode = os.FileMode(0644)

// loggerOnce initialises the process-wide logger exactly once.
// The log file, when opened, lives for the entire process lifetime and is
// intentionally not closed (closing stdout/stderr would be harmful, and a
// file-backed logger is flushed by the OS on process exit).
var loggerOnce = sync.OnceValue(func() *slog.Logger {
	out := openLogOutput()
	opts := &slog.HandlerOptions{Level: resolveLevel()}

	var handler slog.Handler
	switch strings.ToLower(common.EnvOr("LOG_FORMAT", "text")) {
	case "json":
		handler = slog.NewJSONHandler(out, opts)
	default:
		handler = slog.NewTextHandler(out, opts)
	}

	return slog.New(handler)
})

// openLogOutput returns the writer to use for log output.
// When LOG_FILE_PATH is set and the file can be opened, it returns that file;
// otherwise it falls back to os.Stdout and reports the error to os.Stderr.
func openLogOutput() *os.File {
	logFilePath := common.EnvOr("LOG_FILE_PATH", "")
	if logFilePath == "" {
		return os.Stdout
	}

	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, logFileMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log: failed to open %q: %v; falling back to stdout\n", logFilePath, err)
		return os.Stdout
	}
	return f
}

// resolveLevel reads LOG_LEVEL (debug/info/warn/error) and returns the
// corresponding slog.Level, defaulting to Info on unknown values.
func resolveLevel() slog.Level {
	switch strings.ToLower(common.EnvOr("LOG_LEVEL", "info")) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// GetLogger returns the process-wide singleton logger.
func GetLogger() *slog.Logger {
	return loggerOnce()
}
