package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/handlers"
)

func SetupRoutes(app *fiber.App) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Hello, World!",
		})
	})

	app.Get("/health", handlers.HandleHealthCheck)

	// setup the todos group
	todos := app.Group("/todos")
	todos.Get("/", handlers.HandleAllTodos)
	todos.Post("/", handlers.HandleCreateTodo)
	todos.Put("/:id", handlers.HandleUpdateTodo)
	todos.Get("/:id", handlers.HandleGetOneTodo)
	todos.Delete("/:id", handlers.HandleDeleteTodo)
}
