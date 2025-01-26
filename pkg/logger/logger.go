package logger

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
)

type loggerContextKey struct{}

var contextKey loggerContextKey

type logger struct {
	handlers []slog.Handler
}

var globalLogger = &logger{}

func (l *logger) clone() *logger {
	c := *l
	c.handlers = slices.Clip(c.handlers)
	return &c
}

func RegisterHandler(handler slog.Handler) {
	globalLogger.handlers = append(globalLogger.handlers, handler)
}

func EchoWithAttrs(e echo.Context, attrs ...slog.Attr) {
	ctx := WithAttrs(e.Request().Context(), attrs...)
	e.SetRequest(e.Request().WithContext(ctx))
}

func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	logger := loggerFromContext(ctx).clone()
	newHandlers := make([]slog.Handler, len(logger.handlers))
	for i, handler := range logger.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	logger.handlers = newHandlers
	return context.WithValue(ctx, contextKey, logger)
}

func EchoWithGroup(e echo.Context, name string) {
	ctx := WithGroup(e.Request().Context(), name)
	e.SetRequest(e.Request().WithContext(ctx))
}

func WithGroup(ctx context.Context, name string) context.Context {
	logger := loggerFromContext(ctx).clone()
	newHandlers := make([]slog.Handler, len(logger.handlers))
	for i, handler := range logger.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	logger.handlers = newHandlers
	return context.WithValue(ctx, contextKey, logger)
}

func (l *logger) enabled(ctx context.Context, level slog.Level) bool {
	return slog.Default().Enabled(ctx, level)
}

func (l *logger) handle(ctx context.Context, level slog.Level, record slog.Record) {
	if len(l.handlers) == 0 {
		_ = slog.Default().Handler().Handle(ctx, record)
	} else {
		for _, handler := range l.handlers {
			if handler.Enabled(ctx, level) {
				_ = handler.Handle(ctx, record)
			}
		}
	}

}

func loggerFromContext(ctx context.Context) *logger {
	logger, ok := ctx.Value(contextKey).(*logger)
	if !ok {
		return globalLogger
	}
	return logger
}

func logAttrsWrapped(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	logger := loggerFromContext(ctx)
	if !logger.enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip [Callers, logAttrsWrapped, Wrapper]
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.AddAttrs(attrs...)
	logger.handle(ctx, level, r)
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
