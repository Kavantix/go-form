package resources

import (
	"context"
	"fmt"

	. "github.com/Kavantix/go-form/interfaces"
)

type ValidationError struct {
	FieldName string
	Message   string
	Reason    error
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("Validation of field '%s' failed with error: %s", e.FieldName, e.Reason.Error())
}

type ParsingError struct {
	FieldName string
	Reason    error
	Message   string
}

func (e ParsingError) Error() string {
	return fmt.Sprintf("Parsing of field '%s' failed with error: %s", e.FieldName, e.Reason.Error())
}

type Resource[T any] interface {
	Title() string
	FetchPage(ctx context.Context, page, pageSize int) ([]T, error)
	FetchRow(ctx context.Context, id int32) (*T, error)
	ParseRow(ctx context.Context, id *int, formFields map[string]string) (*T, error)
	CreateRow(ctx context.Context, row *T) (int32, error)
	UpdateRow(ctx context.Context, row *T) error
	TableConfig() TableConfig[T]
	FormConfig() FormConfig[T]
	Location(row *T) string
}

func NewResourceTableConfig[T any](resource Resource[T]) TableConfigBuilder[T] {
	return NewTableConfig(func(row T) string { return resource.Location(&row) }).
		WithTitle(resource.Title()).
		WithStreamUrl(fmt.Sprintf("%s/stream", resource.Location(nil))).
		WithCreate(
			fmt.Sprintf("Add %s", resource.Title()),
			fmt.Sprintf("%s/create", resource.Location(nil)),
		)
}
