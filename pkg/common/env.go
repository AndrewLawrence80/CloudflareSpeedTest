package common

import (
	"log"
	"os"
	"strconv"
)

func MustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

func EnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func EnvInt(key string, fallback int) int {
	s := EnvOr(key, "")
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		panic("invalid environment value for " + key + ": " + err.Error())
	}
	return v
}

func EnvUint(key string, fallback uint) uint {
	s := EnvOr(key, "")
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic("invalid environment value for " + key + ": " + err.Error())
	}
	return uint(v)
}

func EnvFloat(key string, fallback float64) float64 {
	s := EnvOr(key, "")
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic("invalid environment value for " + key + ": " + err.Error())
	}
	return v
}
