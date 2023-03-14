package handlers

import (
	"fmt"
	"strings"

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
			Email   string `json:"email"`
			Purpose string `json:"purpose"`
		}
		var input Input
		if err := c.BodyParser(&input); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err.Error())
		}

		linkTokenResp, err := plaidClient.LinkTokenCreate(input.Email, input.Purpose)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to create link token", err.Error())
		}

		CreateCookie(c, fmt.Sprintf("%v_link_token", input.Email), linkTokenResp.Token)
		CreateCookie(c, input.Email, linkTokenResp.UserId)
		id, err := primitive.ObjectIDFromHex(linkTokenResp.UserId)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to get ObjectId from Hex", err.Error())
		}

		plaidClient.SetLinkToken(&models.Token{
			User:  &models.User{ID: id, Email: input.Email},
			Value: linkTokenResp.Token,
		})
		msg := fmt.Sprintf("Successfully received link token from plaid with %+v purpose", input.Purpose)
		return FiberJsonResponse(c, fiber.StatusOK, "success", msg, LinkTokenResponse{Token: linkTokenResp.Token})
	}
}

type Input struct {
	Email       string               `json:"email"`
	PublicToken string               `json:"public_token"`
	Purpose     models.Purpose       `json:"purpose"`
	MetaData    models.PlaidMetaData `json:"meta_data,omitempty"`
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
func ExchangePublicToken(plaidClient *client.PlaidClient) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var input Input
		if err := c.BodyParser(&input); err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to parse input", err.Error())
		}
		if strings.HasPrefix(input.Email, "public") {
			temp := input.Email
			input.Email = input.PublicToken
			input.PublicToken = temp
			plaidClient.L.Info("INPUT: ", input)
		}
		plaidClient.L.Info("METADATA: ", input.MetaData)

		user, err := plaidClient.GetUser(input.Email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to get user for token", err.Error())
		}

		token, err := plaidClient.ExchangePublicToken(plaidClient.C, input.PublicToken)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to exchange for token", err.Error())
		}

		token.User = user
		token.Institution = input.MetaData.Institution.Name
		token.InstitutionID = input.MetaData.Institution.InstitutionId
		token.Purpose = input.Purpose
		plaidClient.L.Info("TOKEN: ", token)

		// dbToken, err := plaidClient.GetUserToken(ctx, user)
		// if err == mongo.ErrNoDocuments || c.Method() == http.MethodPost {
		if err = plaidClient.SaveToken(token); err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to save token", err.Error())
		}

		err = GetandSaveAccountDetails(plaidClient, token, c)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Failure to get and save account details", err.Error())
		}
		// } else {
		// 	if err = plaidClient.UpdateToken(ctx, dbToken.ID, token.Value, token.ItemId); err != nil {
		// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Failure to update access token", "data": err})
		// 	}
		// }
		return FiberJsonResponse(c, fiber.StatusOK, "success", "Access token created successfully", Response{token.Value, token.ItemId, input})
	}
}

func GetandSaveAccountDetails(plaidClient *client.PlaidClient, token *models.Token, c *fiber.Ctx) error {
	accountDetails, err := plaidClient.GetAccountDetails(token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to get account details", "data": err.Error()})
	}

	accounts := accountDetails.Accounts
	plaidAccToDBAccId := make(map[string]string)
	transactions := accountDetails.Transactions

	for _, account := range accounts {
		req := &models.CreateAccountRequest{Account: account}
		dbAccount, err := plaidClient.CreateAccount(plaidClient.C, req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to save account account", "data": err.Error()})
		}
		plaidAccToDBAccId[dbAccount.PlaidAccountId] = dbAccount.ID.Hex()
	}

	if token.Purpose == models.PURPOSE_CREDIT {
		for _, transaction := range transactions {
			trxnID, _ := primitive.ObjectIDFromHex(plaidAccToDBAccId[transaction.PlaidAccountId])
			transaction.AccountId = trxnID
			req := &models.CreateTransactionRequest{Transaction: transaction}
			_, err = plaidClient.CreateTransaction(plaidClient.C, req)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to save account transaction", "data": err.Error()})
			}
		}
	}
	return nil
}

func ArePlaidAccountsLinked(plaidClient *client.PlaidClient) func(c *fiber.Ctx) error {
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

		debitAcc, err := plaidClient.IsDebitAccountLinked(user.GetID())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on fetching user's credit accounts", "data": err.Error()})
		}
		creditAcc, err := plaidClient.IsCreditAccountLinked(user.GetID())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on fetching user's credit accounts", "data": err.Error()})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "data": Exist{debitAcc.Status, creditAcc.Status}})
	}
}
