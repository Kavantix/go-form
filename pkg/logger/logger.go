package logger

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/labstack/echo/v4"
)

func logAttrsWrapped(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	logger := slog.Default()
	if !logger.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip [Callers, logAttrsWrapped, Wrapper]
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.AddAttrs(attrs...)
	_ = logger.Handler().Handle(ctx, r)
}

func Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	logAttrsWrapped(ctx, slog.LevelDebug, msg, attrs...)
}

func EchoDebug(c echo.Context, msg string, attrs ...slog.Attr) {
	logAttrsWrapped(c.Request().Context(), slog.LevelDebug, msg, attrs...)
}

func Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	logAttrsWrapped(ctx, slog.LevelInfo, msg, attrs...)
}

func EchoInfo(c echo.Context, msg string, attrs ...slog.Attr) {
	logAttrsWrapped(c.Request().Context(), slog.LevelInfo, msg, attrs...)
}

func Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	logAttrsWrapped(ctx, slog.LevelWarn, msg, attrs...)
}

func EchoWarn(c echo.Context, msg string, attrs ...slog.Attr) {
	logAttrsWrapped(c.Request().Context(), slog.LevelWarn, msg, attrs...)
}

func Error(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	logAttrsWrapped(ctx, slog.LevelError, fmt.Sprintf("%s: %s", msg, err), attrs...)
}

func EchoError(c echo.Context, msg string, err error, attrs ...slog.Attr) {
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	logAttrsWrapped(c.Request().Context(), slog.LevelError, fmt.Sprintf("%s: %s", msg, err), attrs...)
}

func Critical(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	logAttrsWrapped(ctx, LevelCritical, fmt.Sprintf("%s: %s", msg, err), attrs...)
}

func EchoCritical(c echo.Context, msg string, err error, attrs ...slog.Attr) {
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	logAttrsWrapped(c.Request().Context(), LevelCritical, fmt.Sprintf("%s: %s", msg, err), attrs...)
}
