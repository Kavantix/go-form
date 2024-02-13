package interfaces

import (
	"fmt"
	"os"
	"runtime/debug"
)

type ColumnConfig[T any] struct {
	Label string
	Value func(row T) string
	Url   func(row T) string
}

type tableConfig[T any] struct {
	title       string
	createLabel string
	createUrl   string
	rowUrl      func(row T) string
	columns     []ColumnConfig[T]
	streamUrl   string
}

var _ TableConfig[int] = tableConfig[int]{}

type TableConfig[T any] interface {
	Title() string
	CreateLabel() string
	CreateUrl() string
	RowUrl(row T) string
	Columns() []ColumnConfig[T]
	StreamUrl() string
}

type TableConfigBuilder[T any] interface {
	Build() tableConfig[T]
	WithTitle(title string) TableConfigBuilder[T]
	WithColumns(columns []ColumnConfig[T]) TableConfigBuilder[T]
	WithCreate(label, url string) TableConfigBuilder[T]
	WithStreamUrl(url string) TableConfigBuilder[T]
}

func (c tableConfig[T]) Build() tableConfig[T] {
	if c.rowUrl == nil {
		fmt.Fprintf(os.Stderr, "Row url is not set!\n")
		debug.PrintStack()
	}
	return c
}

func (c tableConfig[T]) Title() string              { return c.title }
func (c tableConfig[T]) CreateLabel() string        { return c.createLabel }
func (c tableConfig[T]) CreateUrl() string          { return c.createUrl }
func (c tableConfig[T]) Columns() []ColumnConfig[T] { return c.columns }
func (c tableConfig[T]) StreamUrl() string          { return c.streamUrl }

func (c tableConfig[T]) RowUrl(row T) string {
	if c.rowUrl == nil {
		fmt.Fprintf(os.Stderr, "Row url is not set!")
		debug.PrintStack()
		return "error"
	}
	return c.rowUrl(row)
}

func (c tableConfig[T]) WithTitle(title string) TableConfigBuilder[T] {
	c.title = title
	return c
}

func (c tableConfig[T]) WithStreamUrl(url string) TableConfigBuilder[T] {
	c.streamUrl = url
	return c
}

func (c tableConfig[T]) WithColumns(columns []ColumnConfig[T]) TableConfigBuilder[T] {
	c.columns = columns
	return c
}

func (c tableConfig[T]) WithCreate(label, url string) TableConfigBuilder[T] {
	c.createLabel = label
	c.createUrl = url
	return c
}

func NewTableConfig[T any](rowUrl func(row T) string) TableConfigBuilder[T] {
	return tableConfig[T]{
		rowUrl: rowUrl,
	}
}
