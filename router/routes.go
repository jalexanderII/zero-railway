package router

import (
	"os"

	client "github.com/jalexanderII/zero-railway/app/clients"

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
	plaidClient := client.NewPlaidClient(os.Getenv("PLAID_COLLECTION"), l)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Hello, World!",
		})
	})

	app.Get("/health", handlers.HandleHealthCheck)

	api := app.Group("/api")
	api.Get("/user/:id", handlers.GetUserByID(userHandler))
	api.Get("/cleanup/:test", handlers.CleanUp(userHandler))

	coreEndpoints := api.Group("/core")
	coreEndpoints.Get("/kpi/:email", handlers.GetKPIs(accountHandler, planningURL))
	coreEndpoints.Get("/paymentplan/:email", handlers.GetPaymentPlans(paymentTaskHandler, planningURL))
	coreEndpoints.Post("/paymentplan/:email", handlers.CreatePaymentPlan(paymentTaskHandler, planningURL))
	coreEndpoints.Delete("/paymentplan/:id", handlers.DeletePaymentPlan(paymentTaskHandler, planningURL))

	accounts := coreEndpoints.Group("/accounts")
	accounts.Get("/:email", handlers.GetUsersAccountsByEmail(accountHandler))
	accounts.Get("/user_id/:user_id", handlers.GetUsersAccountsByUserID(accountHandler))
	accounts.Get("/acc_id/:acc_id", handlers.GetAccount(accountHandler))

	transactions := coreEndpoints.Group("/transactions")
	transactions.Get("/:email", handlers.GetUsersTransactions(transactionHandler))

	paymentTasks := coreEndpoints.Group("/payment_tasks")
	paymentTasks.Get("/:email", handlers.GetUsersPaymentTasks(paymentTaskHandler))

	users := coreEndpoints.Group("/users")
	users.Post("/", handlers.CreateUser(userHandler))
	users.Get("/:email", handlers.GetUser(userHandler))
	users.Put("/:email", handlers.UpdateUserPhone(userHandler))
	clerk := users.Group("/clerk")
	clerk.Post("/", handlers.CreateUserClerkWebhook(userHandler))
	// clerk.Delete("/", handlers.DeleteUserClerkWebhook(userHandler))
	// clerk.Patch("/", handlers.UpdateUserClerkWebhook(userHandler))

	planning := api.Group("/planning")
	planning.Get("/waterfall/:email", handlers.GetWaterfall(accountHandler, planningURL))

	// TODO: Add swagger annotations
	plaidEndpoints := api.Group("/plaid")
	plaidEndpoints.Post("/info", handlers.Info(plaidClient))
	plaidEndpoints.Get("/link/:email/:purpose", handlers.Link)
	plaidEndpoints.Post("/create_link", handlers.CreateLinkToken(plaidClient))
	plaidEndpoints.Post("/exchange", handlers.ExchangePublicToken(plaidClient))
	plaidEndpoints.Get("/linked/:email", handlers.ArePlaidAccountsLinked(plaidClient))

	notificationEndpoints := api.Group("/notify")
	notificationEndpoints.Get("/", handlers.NotifyUsersUpcomingPaymentActions(accountHandler, planningURL))
}
