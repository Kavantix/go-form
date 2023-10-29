package resources

import (
	. "github.com/Kavantix/go-form/interfaces"
)

type ColumnConfig[T any] struct {
	Name  string
	Value func(row *T) string
}

type Resource[T any] interface {
	Title() string
	TableConfig() [](ColumnConfig[T])
	FormConfig() FormConfig[T]
	Location(row *T) string
}
