package main

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

var AppConfig *Config

func LoadConfig() error {
	err := godotenv.Load()
	if err != nil {
		return err
	}

	dolarVzlaKey := os.Getenv("DOLAR_VZLA_API_KEY")
	if dolarVzlaKey == "" {
		return errors.New("Missing DOLAR_VZLA_API_KEY env")
	}

	appPort, err := strconv.ParseUint(os.Getenv("APP_PORT"), 10, 64)
	
	if err != nil {
		// Always default to 8080, even if malformed
		fmt.Println("Missing or malformed APP_PORT. Usin 8080 as default")
		appPort = 8080
	}

	AppConfig = &Config{
		DolarVzlaApiKey: dolarVzlaKey,
		AppPort:         uint(appPort),
	}

	return nil
}
