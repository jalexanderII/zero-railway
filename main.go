package main

import (
	"os"

	"github.com/jalexanderII/zero-railway/app"
)

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	} else {
		port = ":" + port
	}

	return port
}

// @title Zero Fintech Backend API
// @version 0.1
// @description This is the backend API for the Zero Fintech app.
// @contact.name Joel Alexander
// @license.name MIT
// @host localhost:8080
// @BasePath /
func main() {
	err := app.SetupAndRunApp(getPort())
	if err != nil {
		panic(err)
	}
}
