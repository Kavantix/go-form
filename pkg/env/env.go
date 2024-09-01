package env

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type ErrMissingEnv struct {
	MissingKey string
}

func (e ErrMissingEnv) Error() string {
	return fmt.Sprintf("env variable `%s` is missing", e.MissingKey)
}

func LoadDotEnv() error {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to load .env file: %w", err)
	}
	return nil
}

func MustLookup(key string) string {
	value, exists := os.LookupEnv(key)
	value = strings.TrimSpace(value)
	if !exists || value == "" {
		panic(ErrMissingEnv{key})
	}
	return value
}

func Lookup(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	value = strings.TrimSpace(value)
	if !exists || value == "" {
		return fallback
	}
	return value
}
