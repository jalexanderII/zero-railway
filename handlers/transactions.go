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
// @Router /transactions/:email [get]
func GetUsersTransactions(h *Handler, rcache *cache.Cache) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")
		user, err := h.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err.Error())
		}

		transactions, err := FetchTransactionDetails(*user.GetID(), h.P, rcache)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "transactions for that user not found", err.Error())
		}

		//transactions := make([]models.Transaction, 0)
		//filter := bson.M{"user_id": user.ID}
		//opts := options.Find().SetSkip(0).SetLimit(1000)
		//cursor, err := h.Db.Find(h.C, filter, opts)
		//if err != nil {
		//	return FiberJsonResponse(c, fiber.StatusNotFound, "error", "transactions for that user not found", err.Error())
		//}
		//
		//if err = cursor.All(h.C, &transactions); err != nil {
		//	return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed to unmarshall transactions", err.Error())
		//}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "user transactions", transactions)
	}
}
