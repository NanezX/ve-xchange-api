package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     uint
	DatabaseURL string
}

func LoadConfig() (Config, error) {
	err := godotenv.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("malformed env file: %w", err)
		}
		// Continue
	}

	appPort, err := strconv.ParseUint(os.Getenv("APP_PORT"), 10, 64)
	if err != nil {
		slog.Warn("missing or malformed APP_PORT, using default", "default", 8080)
		appPort = 8080
	}

	return Config{
		AppPort:     uint(appPort),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}, nil

}
