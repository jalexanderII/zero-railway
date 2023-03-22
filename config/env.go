package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadENV will load the .env file if the GO_ENV environment variable is not set
func LoadENV() error {
	goEnv := os.Getenv("RAILWAY_ENVIRONMENT")
	log.Println("RAILWAY_ENVIRONMENT: ", goEnv)
	// use local .env file if railway env is local, otherwise use the env vars set in the railway console
	if goEnv != "production" {
		err := godotenv.Load()
		if err != nil {
			return err
		}
	}
	return nil
}
