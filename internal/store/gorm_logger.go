package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// slogGormLogger adapts pkg/log (*slog.Logger) to satisfy gorm/logger.Interface.
type slogGormLogger struct {
	slog          *slog.Logger
	level         gormlogger.LogLevel
	slowThreshold time.Duration
}

func newSlogGormLogger(sl *slog.Logger) gormlogger.Interface {
	return &slogGormLogger{
		slog:          sl,
		level:         gormlogger.Info,
		slowThreshold: 200 * time.Millisecond,
	}
}

func (l *slogGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	copy := *l
	copy.level = level
	return &copy
}

func (l *slogGormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Info {
		l.slog.InfoContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *slogGormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Warn {
		l.slog.WarnContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *slogGormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Error {
		l.slog.ErrorContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *slogGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && l.level >= gormlogger.Error:
		l.slog.ErrorContext(ctx, "gorm query error",
			"error", err,
			"elapsed", elapsed,
			"rows", rows,
			"sql", sql,
		)
	case elapsed > l.slowThreshold && l.slowThreshold > 0 && l.level >= gormlogger.Warn:
		l.slog.WarnContext(ctx, "gorm slow query",
			"elapsed", elapsed,
			"rows", rows,
			"sql", sql,
		)
	case l.level >= gormlogger.Info:
		l.slog.InfoContext(ctx, "gorm query",
			"elapsed", elapsed,
			"rows", rows,
			"sql", sql,
		)
	}
}
