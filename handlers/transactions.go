package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// @Summary Get transactions for a single user.
// @Description fetch all transactions for the user.
// @Tags transactions
// @Param email path string true "User email"
// @Produce json
// @Success 200 {object} []models.Transaction
// @Router /transactions/:email [get]
func GetUsersTransactions(h *Handler) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")
		user, err := h.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err)
		}

		transactions := make([]models.Transaction, 0)
		filter := bson.M{"_id": user.ID}
		opts := options.Find().SetSkip(0).SetLimit(1000)
		cursor, err := h.Db.Find(h.C, filter, opts)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "transactions for that user not found", err)
		}

		if err = cursor.All(h.C, &transactions); err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed to unmarshall transactions", err)
		}
		
		h.L.Info("Transactions found, first 10", transactions[:10])
		return FiberJsonResponse(c, fiber.StatusOK, "success", "user transactions", transactions)
	}
}
