package mails

import (
	"github.com/Kavantix/go-form/database"
	"github.com/matcornic/hermes/v2"
)

type LoginMailContent struct {
	User *database.UserRow
	Link string
}

func Login(content LoginMailContent) *Email {
	return &Email{
		subject: "Your login link to go-form",
		body: hermesBody(hermes.Email{
			Body: hermes.Body{
				Name: content.User.Name,
				Intros: []string{
					"Welcome to go-form! We're very excited to have you on board.",
				},

				Actions: []hermes.Action{
					{
						Instructions: "To get started with go-form, please click here:",
						Button: hermes.Button{
							Color: "#646EE4", // Optional action button color
							Text:  "Login",
							Link:  content.Link,
						},
					},
				},
				Outros: []string{
					"Need help, or have questions? Just reply to this email, we'd love to help.",
				},
			},
		}),
	}
}
