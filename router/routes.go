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
	planningURL := os.Getenv("PLANNING_URL")

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
	api.Get("/user/:id", handlers.GetUserByID(userHandler))

	coreEndpoints := api.Group("/core")
	coreEndpoints.Get("/kpi/:email", handlers.GetKPIs(accountHandler, planningURL))
	//coreEndpoints.Post("/paymentplan/:email",)

	accounts := coreEndpoints.Group("/accounts")
	accounts.Get("/:email", handlers.GetUsersAccountsByEmail(accountHandler))
	accounts.Get("/:user_id", handlers.GetUsersAccountsByUserID(accountHandler))
	accounts.Get("/:acc_id", handlers.GetAccount(accountHandler))

	transactions := coreEndpoints.Group("/transactions")
	transactions.Get("/:email", handlers.GetUsersTransactions(transactionHandler))

	paymentTasks := coreEndpoints.Group("/payment_tasks")
	paymentTasks.Get("/:email", handlers.GetUsersPaymentTasks(paymentTaskHandler))

	users := coreEndpoints.Group("/users")
	users.Post("/", handlers.CreateUser(userHandler))
	users.Get("/:email", handlers.GetUser(userHandler))

	//planning := api.Group("/planning")
	//planning.Get("/waterfall/:email", handlers.GetUsersAccountsByEmail(accountHandler))

	//plaidEndpoints := api.Group("/plaid")
	//plaidEndpoints.Post("/create_link", )
	//plaidEndpoints.Post("/exchange", )
	//plaidEndpoints.Get("/linked/:email", )

	//notificationEndpoints := api.Group("/notify")
	//notificationEndpoints.Post("/", )
}
