package main

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	DolarVzlaApiKey string
}

var AppConfig *Config

func LoadConfig() error {
	err := godotenv.Load()
	if err != nil {
		return err
	}

	AppConfig = &Config{
		DolarVzlaApiKey: os.Getenv("DOLAR_VZLA_API_KEY"),
	}

	return nil
}
