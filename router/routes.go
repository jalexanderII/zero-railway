package router

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/handlers"
	"github.com/sirupsen/logrus"
)

// Create a new instance of the logger.
var l = logrus.New()

func SetupRoutes(app *fiber.App) {

	accountHandler := handlers.NewHandler(os.Getenv("ACCOUNT_COLLECTION"), l)
	transactionHandler := handlers.NewHandler(os.Getenv("TRANSACTION_COLLECTION"), l)
	paymentTaskHandler := handlers.NewHandler(os.Getenv("PAYMENT_TASK_COLLECTION"), l)
	userHandler := handlers.NewHandler(os.Getenv("USER_COLLECTION"), l)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Hello, World!",
		})
	})

	app.Get("/health", handlers.HandleHealthCheck)

	todos := app.Group("/todos")
	todos.Get("/", handlers.HandleAllTodos)
	todos.Post("/", handlers.HandleCreateTodo)
	todos.Put("/:id", handlers.HandleUpdateTodo)
	todos.Get("/:id", handlers.HandleGetOneTodo)
	todos.Delete("/:id", handlers.HandleDeleteTodo)

	api := app.Group("/api")
	coreEndpoints := api.Group("/core")

	accounts := coreEndpoints.Group("/accounts")
	accounts.Get("/:email", handlers.GetUsersAccounts(accountHandler))

	transactions := coreEndpoints.Group("/transactions")
	transactions.Get("/:email", handlers.GetUsersTransactions(transactionHandler))

	paymentTasks := coreEndpoints.Group("/payment_tasks")
	paymentTasks.Get("/:email", handlers.GetUsersPaymentTasks(paymentTaskHandler))

	users := coreEndpoints.Group("/users")
	users.Post("/", handlers.CreateUser(userHandler))
	users.Get("/:email", handlers.GetUser(userHandler))
}
