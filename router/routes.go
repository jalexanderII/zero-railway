package router

import (
	"github.com/go-redis/redis/v8"

	"github.com/go-redis/cache/v8"
	"os"
	"time"

	client "github.com/jalexanderII/zero-railway/app/clients"

	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/handlers"
	"github.com/sirupsen/logrus"
)

// Create a new instance of the logger.
var l = logrus.New()

// SetupRoutes establish all endpoints
func SetupRoutes(app *fiber.App) {
	opt, err := redis.ParseURL(os.Getenv("REDIS_URI"))
	if err != nil {
		panic(err)
	}

	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:       []string{opt.Addr},
		DB:          opt.DB,
		Password:    opt.Password,
		DialTimeout: opt.DialTimeout,
	})
	rcache := cache.New(&cache.Options{
		Redis:      rdb,
		LocalCache: cache.NewTinyLFU(1000, 15*time.Minute),
	})

	plaidClient := client.NewPlaidClient(os.Getenv("PLAID_COLLECTION"), l)
	accountHandler := handlers.NewHandler(os.Getenv("ACCOUNT_COLLECTION"), l, plaidClient)
	transactionHandler := handlers.NewHandler(os.Getenv("TRANSACTION_COLLECTION"), l, plaidClient)
	paymentTaskHandler := handlers.NewHandler(os.Getenv("PAYMENT_TASK_COLLECTION"), l, plaidClient)
	userHandler := handlers.NewHandler(os.Getenv("USER_COLLECTION"), l, plaidClient)
	planningURL := os.Getenv("PLANNING_URL")
	twilioClient := client.NewTwilioClient(l)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Hello, World!",
		})
	})

	app.Get("/health", handlers.HandleHealthCheck)
	app.Get("/clear_cache/:email", handlers.ClearCache(userHandler, rcache))

	api := app.Group("/api")
	api.Get("/user/:id", handlers.GetUserByID(userHandler))
	api.Get("/cleanup/:test", handlers.CleanUp(userHandler))

	coreEndpoints := api.Group("/core")
	coreEndpoints.Get("/kpi/:email", handlers.GetKPIs(accountHandler, planningURL, rcache))
	coreEndpoints.Get("/paymentplan/:email", handlers.GetPaymentPlans(paymentTaskHandler, planningURL, rcache))
	coreEndpoints.Post("/paymentplan/:email", handlers.CreatePaymentPlan(paymentTaskHandler, planningURL, rcache))
	coreEndpoints.Post("/paymentplan/delete/:id", handlers.DeletePaymentPlan(paymentTaskHandler, planningURL))

	accounts := coreEndpoints.Group("/accounts")
	accounts.Get("/:email", handlers.GetUsersAccountsByEmail(accountHandler, rcache))
	accounts.Get("/user_id/:user_id", handlers.GetUsersAccountsByUserID(accountHandler, rcache))
	accounts.Get("/acc_id/:acc_id/:user_id", handlers.GetAccount(accountHandler, rcache))

	transactions := coreEndpoints.Group("/transactions")
	transactions.Get("/:email", handlers.GetUsersTransactions(transactionHandler, rcache))

	paymentTasks := coreEndpoints.Group("/payment_tasks")
	paymentTasks.Get("/:email", handlers.GetUsersPaymentTasks(paymentTaskHandler, rcache))

	users := coreEndpoints.Group("/users")
	users.Post("/", handlers.CreateUser(userHandler, rcache))
	users.Get("/:email", handlers.GetUser(userHandler, rcache))
	users.Put("/:email", handlers.UpdateUserPhone(userHandler, rcache))

	clerk := users.Group("/clerk")
	clerk.Post("/", handlers.CreateUserClerkWebhook(userHandler, rcache))
	// clerk.Delete("/", handlers.DeleteUserClerkWebhook(userHandler))
	// clerk.Patch("/", handlers.UpdateUserClerkWebhook(userHandler))

	planning := api.Group("/planning")
	planning.Get("/waterfall/:email", handlers.GetWaterfall(accountHandler, planningURL, rcache))
	planning.Post("/accept", handlers.AcceptPaymentPlan(paymentTaskHandler, planningURL, rcache))

	// TODO: Add swagger annotations
	plaidEndpoints := api.Group("/plaid")
	plaidEndpoints.Post("/info", handlers.Info(plaidClient))
	plaidEndpoints.Get("/link/:email/:purpose", handlers.Link)
	plaidEndpoints.Post("/create_link", handlers.CreateLinkToken(plaidClient))
	plaidEndpoints.Post("/exchange", handlers.ExchangePublicToken(plaidClient, rcache))
	plaidEndpoints.Get("/linked/:email", handlers.ArePlaidAccountsLinked(plaidClient, rcache))
	plaidEndpoints.Get("/accounts/:email", handlers.GetAccountInfo(plaidClient, rcache))

	notificationEndpoints := api.Group("/notify")
	notificationEndpoints.Get("/", handlers.NotifyUsersUpcomingPaymentActions(twilioClient, accountHandler, planningURL, rcache))
}
