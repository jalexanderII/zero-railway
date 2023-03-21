package handlers

import (
	"github.com/go-redis/cache/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// @Summary Get accounts for a single user.
// @Description fetch all accounts for the user by email.
// @Tags accounts
// @Param email path string true "User email"
// @Produce json
// @Success 200 {object} []models.Account
// @Router /accounts/:email [get]
func GetUsersAccountsByEmail(h *Handler, rcache *cache.Cache) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")

		user, err := h.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err.Error())
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
// @Router /accounts/acc_id/:acc_id/:email [get]
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

		for _, acc := range Accounts {
			if acc.ID == accId {
				return FiberJsonResponse(c, fiber.StatusOK, "success", "account", acc)
			}
		}

		return FiberJsonResponse(c, fiber.StatusNotFound, "error", "account not found", err.Error())
		//
		//
		//var account models.Account
		//filter := bson.M{"_id": accId}
		//if err = h.Db.FindOne(h.C, filter).Decode(&account); err != nil {
		//	return FiberJsonResponse(c, fiber.StatusNotFound, "error", "account not found", err.Error())
		//}
		//return FiberJsonResponse(c, fiber.StatusOK, "success", "account", account)
	}
}

func GetUserAccounts(h *Handler, userId *primitive.ObjectID, rcache *cache.Cache) ([]*models.Account, error) {
	Accounts, err := FetchAccountDetails(*userId, h.P, rcache)
	if err != nil {
		return nil, err
	}
	return Accounts, nil

	//accounts := make([]models.Account, 0)
	//filter := bson.M{"user_id": userId}
	//opts := options.Find().SetSkip(0).SetLimit(1000)
	//cursor, err := h.Db.Find(h.C, filter, opts)
	//if err != nil {
	//	return nil, err
	//}
	//
	//if err = cursor.All(h.C, &accounts); err != nil {
	//	return nil, err
	//}
	//return accounts, nil
}

func GetDebitAccountBalance(h *Handler, userId *primitive.ObjectID, rcache *cache.Cache) *models.GetDebitAccountBalanceResponse {
	//var account models.Account
	//filter := []bson.M{{"user_id": userId}, {"type": "depository"}}
	//err := h.Db.FindOne(h.C, bson.M{"$and": filter}).Decode(&account)
	//if err != nil {
	//	return nil
	//}

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
	//
	//
	//return &models.GetDebitAccountBalanceResponse{
	//	AvailableBalance: account.AvailableBalance,
	//	CurrentBalance:   account.CurrentBalance,
	//}
	return nil
}
