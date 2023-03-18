package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

// AddSwaggerRoutes will add auto generated swagger routes
func AddSwaggerRoutes(app *fiber.App) {
	// setup swagger
	app.Get("/swagger/*", swagger.HandlerDefault)
}
