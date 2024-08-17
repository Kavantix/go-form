package main

import (
	"fmt"
	"strconv"

	"github.com/Kavantix/go-form/templates"
	"github.com/a-h/templ"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

func templateInLayout(c *gin.Context, code int, currentTab string, children ...templ.Component) {
	template(c, code, templates.Layout(currentTab, children...))
}

func template(c *gin.Context, code int, t templ.Component) {
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Status(code)
	span := sentry.StartSpan(
		c.Request.Context(),
		"template.render",
		sentry.WithDescription(fmt.Sprintf("Template: %+v", t)),
	)
	err := t.Render(c.Request.Context(), c.Writer)
	span.Finish()
	if err != nil {
		c.AbortWithError(500, fmt.Errorf("failed to render template `%+v`: %w", t, err))
	}
}

func startSseStream(c *gin.Context) {
	c.Writer.Header().Set("Cache-Control", "no-store")
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Status(200)
}

func sendSseEvent(c *gin.Context, name string, data string) {
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", name, data)
	c.Writer.Flush()
}

func templateEvent(c *gin.Context, name string, t templ.Component) {
	fmt.Fprintf(c.Writer, "event: %s\ndata: ", name)
	span := sentry.StartSpan(
		c.Request.Context(),
		"template.render",
		sentry.WithDescription(fmt.Sprintf("Template: %+v", t)),
	)
	t.Render(c.Request.Context(), c.Writer)
	span.Finish()
	fmt.Fprint(c.Writer, "\n\n")
	c.Writer.Flush()
}

func htmxRedirect(c *gin.Context, location string) {
	c.Header("HX-Location", location)
	c.Status(204)
}

func paginationParams(c *gin.Context) (page, pageSize int, err error) {
	page, err = strconv.Atoi(c.DefaultQuery("page", "0"))
	if err != nil {
		return
	}
	pageSize, err = strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if err != nil {
		return
	}
	return
}
