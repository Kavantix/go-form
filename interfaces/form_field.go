package interfaces

import "github.com/a-h/templ"

type FormConfig[T any] struct {
	SaveUrl func(row *T) string
	Fields  [](FormField[T])
}

type FormField[T any] interface {
	Name() string
	RenderFormField(form FormConfig[T], value *T, validationError string) templ.Component
}
