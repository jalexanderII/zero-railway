package handlers

import (
	"fmt"
	"strings"

	"github.com/go-redis/cache/v8"
	client "github.com/jalexanderII/zero-railway/app/clients"
	"github.com/jalexanderII/zero-railway/models"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Info(plaidClient *client.PlaidClient) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"item_id":      "",
			"access_token": "",
			"products":     plaidClient.Products,
		})
	}
}

// Link will call CreateLinkToken to get a link token, and then call ExchangePublicToken to get an access token
// will be saved to db along with account and transaction details upon success
func Link(c *fiber.Ctx) error {
	return c.Render("index", fiber.Map{
		"Email":   c.Params("email"),
		"Purpose": c.Params("purpose"),
	})
}

func CreateLinkToken(plaidClient *client.PlaidClient) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		type LinkTokenResponse struct {
			Token string `json:"link_token"`
		}

		type Input struct {
			UserId  string `json:"user_id"`
			Purpose string `json:"purpose"`
		}
		var input Input
		if err := c.BodyParser(&input); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err.Error())
		}
		user, err := plaidClient.GetUserByClarkId(input.UserId)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to create link token", err.Error())
		}

		linkTokenResp, err := plaidClient.LinkTokenCreate(user.Email, input.Purpose)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to create link token", err.Error())
		}

		CreateCookie(c, fmt.Sprintf("%v_link_token", user.Email), linkTokenResp.Token)
		CreateCookie(c, user.Email, linkTokenResp.UserId)
		id, err := primitive.ObjectIDFromHex(linkTokenResp.UserId)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to get ObjectId from Hex", err.Error())
		}

		plaidClient.SetLinkToken(&models.Token{
			User:  &models.User{ID: id, Email: user.Email},
			Value: linkTokenResp.Token,
		})
		msg := fmt.Sprintf("Successfully received link token from plaid with %+v purpose", input.Purpose)
		return FiberJsonResponse(c, fiber.StatusOK, "success", msg, LinkTokenResponse{Token: linkTokenResp.Token})
	}
}

type Input struct {
	UserId      string                  `json:"user_id"`
	PublicToken string                  `json:"public_token"`
	Purpose     models.Purpose          `json:"purpose"`
	Institution models.PlaidInstitution `json:"institution,omitempty"`
}

type Response struct {
	AccessToken string `json:"access_token"`
	ItemId      string `json:"item_id"`
	Token       Input  `json:"token"`
}

// @Summary Exchange public token and save account info.
// @Description Save account info from plaid link
// @Tags plaid
// @Accept json
// @Param input body Input true "Input data"
// @Produce json
// @Success 200 {object} Response
// @Router /exchange [post]
func ExchangePublicToken(plaidClient *client.PlaidClient, rcache *cache.Cache) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var input Input
		if err := c.BodyParser(&input); err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to parse input", err.Error())
		}
		if strings.HasPrefix(input.UserId, "public") {
			temp := input.UserId
			input.UserId = input.PublicToken
			input.PublicToken = temp
			plaidClient.L.Info("INPUT: ", input)
		}
		plaidClient.L.Info("METADATA DATA: ", input.Institution)

		user, err := plaidClient.GetUserByClarkId(input.UserId)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to get user for token", err.Error())
		}

		token, err := plaidClient.ExchangePublicToken(plaidClient.C, input.PublicToken)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to exchange for token", err.Error())
		}

		token.User = user
		token.Institution = input.Institution.Name
		token.InstitutionID = input.Institution.InstitutionId
		token.Purpose = input.Purpose
		plaidClient.L.Info("TOKEN: ", token)

		if err = plaidClient.SaveToken(token); err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to save token", err.Error())
		}

		err = GetandSaveAccountDetails(plaidClient, token, c, rcache)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to get and save account details", err.Error())
		}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "Access token created successfully", Response{token.Value, token.ItemId, input})
	}
}

// @Summary Get all account and transaction info for all of a users linked accounts.
// @Description Get all account and transaction info
// @Tags plaid
// @Accept json
// @Param email path string true "User email"
// @Produce json
// @Success 200 {object} models.AccountDetailsResponse
// @Router /accounts/:email [get]
func GetAccountInfo(plaidClient *client.PlaidClient, rcache *cache.Cache) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")
		user, err := plaidClient.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err.Error())
		}

		AccountDetails, err := FetchDataAndCache(*user.GetID(), plaidClient, rcache, false)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to get account details", "data": err.Error()})
		}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "Fetched all account details from cache", AccountDetails)
	}
}

func GetandSaveAccountDetails(plaidClient *client.PlaidClient, token *models.Token, c *fiber.Ctx, rcache *cache.Cache) error {
	_, err := FetchDataAndCache(token.User.ID, plaidClient, rcache, true)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to get account details", "data": err.Error()})
	}
	return nil
}

func ArePlaidAccountsLinked(plaidClient *client.PlaidClient, rcache *cache.Cache) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")
		user, err := plaidClient.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err.Error())
		}

		type Exist struct {
			Debit  bool `json:"debit"`
			Credit bool `json:"credit"`
		}

		debitAcc, err := IsDebitAccountLinked(user.GetID(), plaidClient, rcache)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on fetching user's credit accounts", "data": err.Error()})
		}
		creditAcc, err := IsCreditAccountLinked(user.GetID(), plaidClient, rcache)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on fetching user's credit accounts", "data": err.Error()})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "data": Exist{debitAcc.Status, creditAcc.Status}})
	}
}

func IsDebitAccountLinked(userId *primitive.ObjectID, p *client.PlaidClient, rcache *cache.Cache) (*models.IsAccountLinkedResponse, error) {
	return AccountLinked(userId, p, rcache, "depository")
}

func IsCreditAccountLinked(userId *primitive.ObjectID, p *client.PlaidClient, rcache *cache.Cache) (*models.IsAccountLinkedResponse, error) {
	return AccountLinked(userId, p, rcache, "credit")
}

func AccountLinked(userId *primitive.ObjectID, p *client.PlaidClient, rcache *cache.Cache, accType string) (*models.IsAccountLinkedResponse, error) {
	Accounts, err := FetchAccountDetails(*userId, p, rcache)
	if err != nil {
		return nil, err
	}
	for _, acc := range Accounts {
		if acc.Type == accType {
			return &models.IsAccountLinkedResponse{Status: true}, nil
		}
	}
	return &models.IsAccountLinkedResponse{Status: false}, nil
}
