package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort uint
}

func LoadConfig() (Config, error) {
	err := godotenv.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("Malformed env file: %w", err)
		}
		// Continue
	}

	appPort, err := strconv.ParseUint(os.Getenv("APP_PORT"), 10, 64)
	if err != nil {
		fmt.Println("Missing or malformed APP_PORT. Usin 8080 as default")
		appPort = 8080
	}

	return Config{
		AppPort: uint(appPort),
	}, nil

}
