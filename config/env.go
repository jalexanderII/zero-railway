package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

// LoadENV will load the .env file if the GO_ENV environment variable is not set
func LoadENV() error {
	goEnv := os.Getenv("RAILWAY_ENVIRONMENT")
	fmt.Println("RAILWAY_ENVIRONMENT: ", goEnv)
	// use local .env file if railway env is local, otherwise use the env vars set in the railway console
	if goEnv == "" || goEnv == "local" {
		err := godotenv.Load()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetEnv func to get env values
func GetEnv(key string) string {
	_, b, _, _ := runtime.Caller(0)
	// Root folder of this project
	Root := filepath.Join(filepath.Dir(b), "../")
	environmentPath := filepath.Join(Root, ".env")
	err := godotenv.Load(environmentPath)
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	return os.Getenv(key)
}
