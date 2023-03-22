package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/config"
	"github.com/jalexanderII/zero-railway/database"
	"github.com/jalexanderII/zero-railway/router"
)

// SetupAndRunApp handle app and database start and graceful shutdown
func SetupAndRunApp(port string) error {
	// start database
	err := database.StartMongoDB()
	if err != nil {
		return err
	}

	// defer closing database
	defer database.CloseMongoDB()

	// create app
	app := fiber.New()

	// attach middleware
	FiberMiddleware(app)

	// setup routes
	router.SetupRoutes(app)

	// attach swagger
	config.AddSwaggerRoutes(app)

	StartServerWithGracefulShutdown(app, port)

	return nil
}
