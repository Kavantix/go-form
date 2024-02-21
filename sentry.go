package main

import (
	"fmt"

	"github.com/Kavantix/go-form/templates"
	"github.com/getsentry/sentry-go"
)

func InitSentry() error {
	templates.FrontendSentryDSN = MustLookupEnv("FRONTEND_SENTRY_DSN")
	err := sentry.Init(sentry.ClientOptions{
		Dsn:                MustLookupEnv("SENTRY_DSN"),
		TracesSampleRate:   1.0,
		EnableTracing:      true,
		ProfilesSampleRate: 1.0,
		Environment:        "local",
	})
	if err != nil {
		return fmt.Errorf("sentry.Init: %s", err)
	}
	return nil
}
