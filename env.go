package main

import (
	"log"
	"os"
	"strings"
)

func MustLookupEnv(key string) string {
	value, exists := os.LookupEnv(key)
	value = strings.TrimSpace(value)
	if !exists || value == "" {
		log.Fatalf("Env variable '%s' is required", key)
	}
	return value
}

func LookupEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	value = strings.TrimSpace(value)
	if !exists || value == "" {
		return fallback
	}
	return value
}
