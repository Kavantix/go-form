package main

import (
	"encoding/json"
	"fmt"
	"github.com/Kavantix/go-form/templates"
	"github.com/a-h/templ"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
)

func templateInLayout(c echo.Context, code int, currentTab string, children ...templ.Component) error {
	return template(c, code, templates.Layout(currentTab, children...))
}

type toastVariant int

const (
	ToastInfo    toastVariant = 0
	ToastSuccess toastVariant = 1
	ToastError   toastVariant = 2
)

func (v toastVariant) String() string {
	switch v {
	case ToastError:
		return "error"
	case ToastSuccess:
		return "success"
	default:
		return "info"
	}
}

type ToastConfig struct {
	Message    string
	DurationMs int
	Variant    toastVariant
}

type ToastMessage string

func triggerToast(c echo.Context, toast ToastConfig) error {
	showToast := map[string]any{
		"target":  "#toast-container",
		"message": toast.Message,
		"variant": toast.Variant.String(),
	}
	if toast.DurationMs > 0 {
		showToast["durationMs"] = toast.DurationMs
	}
	jsonToast, err := json.Marshal(map[string]any{
		"show-toast": showToast,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal toast `%+v`: %w", toast, err)
	}
	c.Response().Header().Set("HX-Trigger", string(jsonToast))
	return nil
}

func template(c echo.Context, code int, templates ...templ.Component) error {
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Response().WriteHeader(code)
	for _, t := range templates {
		span := sentry.StartSpan(
			c.Request().Context(),
			"template.render",
			sentry.WithDescription(fmt.Sprintf("Template: %+v", t)),
		)
		err := t.Render(c.Request().Context(), c.Response().Writer)
		span.Finish()
		if err != nil {
			return fmt.Errorf("failed to render template `%+v`: %w", t, err)
		}
	}
	return nil
}

func startSseStream(c echo.Context) {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().WriteHeader(200)
}

func sendSseEvent(c echo.Context, name string, data string) {
	fmt.Fprintf(c.Response().Writer, "event: %s\ndata: %s\n\n", name, data)
	c.Response().Flush()
}

func templateEvent(c echo.Context, name string, t templ.Component) {
	fmt.Fprintf(c.Response().Writer, "event: %s\ndata: ", name)
	span := sentry.StartSpan(
		c.Request().Context(),
		"template.render",
		sentry.WithDescription(fmt.Sprintf("Template: %+v", t)),
	)
	t.Render(c.Request().Context(), c.Response().Writer)
	span.Finish()
	fmt.Fprint(c.Response().Writer, "\n\n")
	c.Response().Flush()
}

func htmxRedirect(e echo.Context, location string) error {
	e.Response().Header().Set("HX-Location", location)
	return e.NoContent(204)
}

func paginationParams(c echo.Context) (page, pageSize int, err error) {
	page = 0
	pageSize = 20
	err = echo.QueryParamsBinder(c).
		Int("page", &page).
		Int("pageSize", &pageSize).
		BindError()

	if err != nil {
		return
	}
	return
}
