package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort         uint
	DolarVzlaApiKey string
}

func LoadConfig() (Config, error) {
	err := godotenv.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("Malformed env file: %w", err)
		}
		// Continue
	}

	dolarVzlaKey := os.Getenv("DOLAR_VZLA_API_KEY")
	if dolarVzlaKey == "" {
		return Config{}, errors.New("Missing DOLAR_VZLA_API_KEY env")
	}

	appPort, err := strconv.ParseUint(os.Getenv("APP_PORT"), 10, 64)
	if err != nil {
		fmt.Println("Missing or malformed APP_PORT. Usin 8080 as default")
		appPort = 8080
	}

	return Config{
		DolarVzlaApiKey: dolarVzlaKey,
		AppPort:         uint(appPort),
	}, nil

}
