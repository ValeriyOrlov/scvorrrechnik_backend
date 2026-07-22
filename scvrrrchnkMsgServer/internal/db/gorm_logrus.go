package db

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"
)

type GormLogrusAdapter struct {
	Logger *logrus.Logger
	Level  logger.LogLevel
}

func (l *GormLogrusAdapter) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.Level = level
	return &newLogger
}

func (l *GormLogrusAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.Level >= logger.Info {
		l.Logger.Infof(msg, data...)
	}
}

func (l *GormLogrusAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.Level >= logger.Warn {
		l.Logger.Warnf(msg, data...)
	}
}

func (l *GormLogrusAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.Level >= logger.Error {
		l.Logger.Errorf(msg, data...)
	}
}

func (l *GormLogrusAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.Level <= logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := logrus.Fields{
		"latency_ms": float64(elapsed.Nanoseconds()) / 1e6,
		"rows":       rows,
		"sql":        sql,
	}
	if err != nil {
		fields["error"] = err
		l.Logger.WithFields(fields).Error("gorm query failed")
	} else if elapsed > 200*time.Millisecond {
		fields["slow_query"] = true
		l.Logger.WithFields(fields).Warn("slow gorm query")
	} else {
		l.Logger.WithFields(fields).Debug("gorm query")
	}
}
