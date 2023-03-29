package handlers

import (
	"github.com/go-redis/cache/v8"
	"github.com/gofiber/fiber/v2"
)

// @Summary Get transactions for a single user.
// @Description fetch all transactions for the user.
// @Tags transactions
// @Param email path string true "User email"
// @Produce json
// @Success 200 {object} []models.Transaction
// @Router /transactions [get]
func GetUsersTransactions(h *Handler, rcache *cache.Cache) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		user, err := GetUserFromCache(c, rcache)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed getting user's account", err.Error())
		}

		transactions, err := FetchTransactionDetails(*user.GetID(), h.P, rcache)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "transactions for that user not found", err.Error())
		}

		return FiberJsonResponse(c, fiber.StatusOK, "success", "user transactions", transactions)
	}
}
