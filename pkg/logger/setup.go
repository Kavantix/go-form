package logger

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Extra log level supported by Cloud Logging
const (
	LevelCritical = slog.Level(12)
)

// Middleware that adds the Cloud Trace ID to the context
// This is used to correlate the structured logs with the Cloud Run
// request log.
func withCloudTraceContext(projectId string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var trace string
			traceHeader := r.Header.Get("X-Cloud-Trace-Context")
			traceParts := strings.Split(traceHeader, "/")
			if len(traceParts) > 0 && len(traceParts[0]) > 0 {
				trace = fmt.Sprintf("projects/%s/traces/%s", projectId, traceParts[0])
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "trace", trace)))
		})
	}
}

func traceFromContext(ctx context.Context) string {
	trace := ctx.Value("trace")
	if trace == nil {
		return ""
	}
	return trace.(string)
}

// Handler that outputs JSON understood by the structured log agent.
// See https://cloud.google.com/logging/docs/agent/logging/configuration#special-fields
type CloudLoggingHandler struct{ handler slog.Handler }

func newCloudLoggingHandler() *CloudLoggingHandler {
	return &CloudLoggingHandler{handler: slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				a.Key = "message"
			} else if a.Key == slog.SourceKey {
				a.Key = "logging.googleapis.com/sourceLocation"
			} else if a.Key == slog.LevelKey {
				a.Key = "severity"
				level := a.Value.Any().(slog.Level)
				if level == LevelCritical {
					a.Value = slog.StringValue("CRITICAL")
				}
			}
			return a
		},
	})}
}

func (h *CloudLoggingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func recordContainsAttr(rec slog.Record, key string) bool {
	result := false
	rec.Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			result = true
			return false
		}
		return true
	})
	return result
}

func recordContainsAttrWithValue(rec slog.Record, key string, value string) bool {
	result := ""
	rec.Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			result = a.Value.String()
			return false
		}
		return true
	})
	return result == value
}

func (h *CloudLoggingHandler) Handle(ctx context.Context, rec slog.Record) error {
	trace := traceFromContext(ctx)
	if trace != "" {
		rec = rec.Clone()
		// Add trace ID	to the record so it is correlated with the Cloud Run request log
		// See https://cloud.google.com/trace/docs/trace-log-integration
		rec.Add("logging.googleapis.com/trace", slog.StringValue(trace))
	}
	if recordContainsAttr(rec, "error") {
		stack := debug.Stack()
		rec.AddAttrs(
			slog.String("exception", fmt.Sprintf("%s:\n%s", rec.Message, string(stack))),
		)
	}
	return h.handler.Handle(ctx, rec)
}

func (h *CloudLoggingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CloudLoggingHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *CloudLoggingHandler) WithGroup(name string) slog.Handler {
	return &CloudLoggingHandler{handler: h.handler.WithGroup(name)}
}

var didInitGoogleCloud = false

func InitGoogleCloudLogger() {
	if didInitGoogleCloud {
		panic("double call to InitGoogleCloudLogger")
	}
	logger := slog.New(newCloudLoggingHandler())
	slog.SetDefault(logger)
	didInitGoogleCloud = true
}

func SetupEchoGoogleCloudLogger(e *echo.Echo, projectId string) {
	if !didInitGoogleCloud {
		panic("cannot setup echo google cloud logger, InitGoogleCloudLogger should be called first")
	}
	logger := slog.Default()
	e.Use(echo.WrapMiddleware(withCloudTraceContext(projectId)))
	recoverConfig := middleware.DefaultRecoverConfig
	recoverConfig.LogErrorFunc = func(c echo.Context, err error, stack []byte) error {
		logger.LogAttrs(c.Request().Context(), LevelCritical, fmt.Sprintf("Recovered from panic: %s", err),
			slog.String("uri", c.Request().RequestURI),
			slog.String("error", err.Error()),
			slog.String("stack_trace", string(stack)),
		)
		return err
	}
	e.Use(middleware.RecoverWithConfig(recoverConfig))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		HandleError: false, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error != nil && v.Status != 404 {
				logger.LogAttrs(c.Request().Context(), slog.LevelError, fmt.Sprintf("Request failed: %s", v.Error),
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("error", v.Error.Error()),
				)
			} else {
				logger.LogAttrs(c.Request().Context(), slog.LevelInfo, "Request successful",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
				)
			}
			return nil
		},
	}))
}
