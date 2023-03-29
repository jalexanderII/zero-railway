package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"time"
)

// FiberMiddleware provides Fiber's built-in middlewares.
// See: https://docs.gofiber.io/api/middleware
func FiberMiddleware(a *fiber.App) {
	a.Use(
		// Add simple logger.
		logger.New(logger.Config{
			Format: "[${ip}]:${port} ${status} - ${method} ${path} ${latency}\n",
		}),
		// Add CORS to each route.
		cors.New(),
		// Add Cache
		cache.New(),
		// add rate limiter
		limiter.New(limiter.Config{
			Max:               20,
			Expiration:        30 * time.Second,
			LimiterMiddleware: limiter.SlidingWindow{},
		}),
		// recover from panic
		recover.New(),
	)
}
