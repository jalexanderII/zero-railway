package handlers

import (
	"github.com/go-redis/cache/v8"
	"github.com/gofiber/fiber/v2"
)

// @Summary Show the status of server.
// @Description get the status of server.
// @Tags health
// @Accept */*
// @Produce plain
// @Success 200 "OK"
// @Router /health [get]
func HandleHealthCheck(c *fiber.Ctx) error {
	return c.SendString("OK")
}

func ClearCache(h *Handler, rcache *cache.Cache) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")

		user, err := h.GetUserByEmail(email, rcache)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed getting users account", err.Error())
		}

		err = rcache.Delete(c.Context(), user.GetID().Hex())
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed clearing cache", err.Error())
		}

		return FiberJsonResponse(c, fiber.StatusOK, "success", "cache cleared", nil)

	}
}
