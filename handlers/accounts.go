package handlers

import (
	"github.com/go-redis/cache/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// @Summary Get accounts for a single user.
// @Description fetch all accounts for the user.
// @Tags accounts
// @Produce json
// @Success 200 {object} []models.Account
// @Router /accounts [get]
func GetUsersAccountsByEmail(h *Handler, rcache *cache.Cache) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		user, err := GetUserFromCache(c, rcache)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed getting user's account", err.Error())
		}
		accounts, err := GetUserAccounts(h, user.GetID(), rcache)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed getting users accounts", err.Error())
		}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "user accounts", accounts)
	}
}

// @Summary Get accounts for a single user.
// @Description fetch all accounts for the user by user id.
// @Tags accounts
// @Param user_id path string true "User ID"
// @Produce json
// @Success 200 {object} []models.Account
// @Router /accounts/:user_id [get]
func GetUsersAccountsByUserID(h *Handler, rcache *cache.Cache) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId, err := primitive.ObjectIDFromHex(c.Params("user_id"))
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusBadRequest, "error", "invalid user id", err.Error())
		}
		accounts, err := GetUserAccounts(h, &userId, rcache)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed getting users accounts", err.Error())
		}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "user accounts", accounts)
	}
}

// @Summary Get a single account
// @Description fetch account by account id.
// @Tags accounts
// @Accept */*
// @Produce json
// @Success 200 {object} models.Account
// @Router /accounts/acc_id/:acc_id/:user_id [get]
func GetAccount(h *Handler, rcache *cache.Cache) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId, err := primitive.ObjectIDFromHex(c.Params("user_id"))
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusBadRequest, "error", "invalid user id", err.Error())
		}

		accId := c.Params("acc_id")
		Accounts, err := FetchAccountDetails(userId, h.P, rcache)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed getting users account", err.Error())
		}
		h.L.Info("accounts fetched", Accounts)

		for _, acc := range Accounts {
			if acc.ID == accId {
				return FiberJsonResponse(c, fiber.StatusOK, "success", "account", acc)
			}
		}

		return FiberJsonResponse(c, fiber.StatusNotFound, "error", "account not found", err.Error())
	}
}

func GetUserAccounts(h *Handler, userId *primitive.ObjectID, rcache *cache.Cache) ([]*models.Account, error) {
	Accounts, err := FetchAccountDetails(*userId, h.P, rcache)
	if err != nil {
		return nil, err
	}
	return Accounts, nil
}

func GetDebitAccountBalance(h *Handler, userId *primitive.ObjectID, rcache *cache.Cache) *models.GetDebitAccountBalanceResponse {
	Accounts, err := FetchAccountDetails(*userId, h.P, rcache)
	if err != nil {
		return nil
	}

	for _, acc := range Accounts {
		if acc.Type == "depository" {
			return &models.GetDebitAccountBalanceResponse{
				AvailableBalance: acc.AvailableBalance,
				CurrentBalance:   acc.CurrentBalance,
			}
		}
	}
	return nil
}
