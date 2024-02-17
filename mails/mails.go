package mails

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/matcornic/hermes/v2"
	"github.com/wneessen/go-mail"
)

type Email struct {
	subject string
	body    struct {
		plainText string
		html      string
	}
}

var mailClient *mail.Client

func Init(mailhogHost string) error {
	var err error
	mailClient, err = mail.NewClient(
		mailhogHost,
		// mail.WithPort(25),
		mail.WithPort(1025),
		// mail.WithSMTPAuth(mail.SMTPAuthPlain),
		// mail.WithUsername("my_username"),
		// mail.WithPassword("extremely_secret_pass"),
		mail.WithTLSPolicy(mail.NoTLS),
	)
	if err != nil {
		return fmt.Errorf("failed to create mail client: %w", err)
	}
	return nil
}

func (message *Email) SendTo(ctx context.Context, toEmail string, ccEmails ...string) error {
	span := sentry.StartSpan(ctx, "function", sentry.WithDescription("Send email"))
	defer span.Finish()
	m := mail.NewMsg()
	if err := m.From("noreply@go-form.test"); err != nil {
		log.Fatalf("failed to set From address: %s", err)
	}
	if err := m.To(toEmail); err != nil {
		log.Fatalf("failed to set To address: %s", err)
	}
	if len(ccEmails) > 0 {
		if err := m.Cc(ccEmails...); err != nil {
			log.Fatalf("failed to set Cc address: %s", err)
		}
	}
	m.Subject(message.subject)
	m.SetBodyString(mail.TypeTextPlain, message.body.plainText)
	m.AddAlternativeString(mail.TypeTextHTML, message.body.html)
	if err := mailClient.DialAndSend(m); err != nil {
		span.Status = sentry.SpanStatusFailedPrecondition
		return fmt.Errorf("failed to send mail: %w", err)
	}

	return nil
}

// Configure hermes by setting a theme and your product info
var h = &hermes.Hermes{
	// Optional Theme
	// Theme: new(Default)
	Product: hermes.Product{
		// Appears in header & footer of e-mails
		Name:      "go-form",
		Link:      "http://go-form.test/",
		Copyright: fmt.Sprintf("Copyright © %v Pieter van Loon. All rights reserved.", time.Now().Year()),
	},
}

func hermesBody(email hermes.Email) struct{ plainText, html string } {
	textBody, err := h.GeneratePlainText(email)
	if err != nil {
		// The only thing that could fail would be configuration error
		// so ok to panic
		panic(fmt.Errorf("Failed to generate text body for email `%+v` with error: %w", email, err))
	}
	htmlBody, err := h.GenerateHTML(email)
	if err != nil {
		// The only thing that could fail would be configuration error
		// so ok to panic
		panic(fmt.Errorf("Failed to generate html body for email `%+v` with error: %w", email, err))
	}
	return struct {
		plainText string
		html      string
	}{
		plainText: textBody,
		html:      htmlBody,
	}
}
