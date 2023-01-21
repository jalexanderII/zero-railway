package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// @Summary Get accounts for a single user.
// @Description fetch all accounts for the user.
// @Tags accounts
// @Param email path string true "User email"
// @Produce json
// @Success 200 {object} []models.Account
// @Router /accounts/:email [get]
func GetUsersAccounts(h *Handler) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")
		user, err := h.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err)
		}

		accounts := make([]models.Account, 0)
		filter := bson.M{"user_id": user.ID}
		opts := options.Find().SetSkip(0).SetLimit(1000)
		cursor, err := h.Db.Find(h.C, filter, opts)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "accounts for that user not found", err)
		}

		if err = cursor.All(h.C, &accounts); err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed to unmarshall accounts", err)
		}
		h.L.Info("Accounts found, first 10", accounts[:10])
		return FiberJsonResponse(c, fiber.StatusOK, "success", "user accounts", accounts)
	}
}
