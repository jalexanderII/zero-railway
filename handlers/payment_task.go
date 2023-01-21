package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// @Summary Get payment_tasks for a single user.
// @Description fetch all payment_tasks for the user.
// @Tags payment_tasks
// @Param email path string true "User email"
// @Produce json
// @Success 200 {object} []models.PaymentTask
// @Router /payment_tasks/:email [get]
func GetUsersPaymentTasks(h *Handler) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")
		user, err := h.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err)
		}

		paymentTasks := make([]models.PaymentTask, 0)
		filter := bson.M{"user_id": user.ID}
		opts := options.Find().SetSkip(0).SetLimit(1000)
		cursor, err := h.Db.Find(h.C, filter, opts)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "payment tasks for that user not found", err)
		}

		if err = cursor.All(h.C, &paymentTasks); err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed to unmarshall payment tasks", err)
		}

		h.L.Info("payment tasks found, first 10", paymentTasks[:10])
		return FiberJsonResponse(c, fiber.StatusOK, "success", "user payment tasks", paymentTasks)
	}
}
