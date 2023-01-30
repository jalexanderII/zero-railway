package handlers

import (
	"fmt"
	client "github.com/jalexanderII/zero-railway/app/clients"
	"github.com/jalexanderII/zero-railway/models"
	"strings"

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
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to create link token", "data": err})
		}

		CreateCookie(c, fmt.Sprintf("%v_link_token", input.Email), linkTokenResp.Token)
		CreateCookie(c, input.Email, linkTokenResp.UserId)
		id, err := primitive.ObjectIDFromHex(linkTokenResp.UserId)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to get ObjectId from Hex", "data": err})
		}

		plaidClient.SetLinkToken(&models.Token{
			User:  &models.User{ID: id, Email: input.Email},
			Value: linkTokenResp.Token,
		})

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "message": fmt.Sprintf("Successfully received link token from plaid with %+v purpose", input.Purpose), "link_token": linkTokenResp.Token})
	}
}

func ExchangePublicToken(plaidClient *client.PlaidClient) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		type Input struct {
			Email       string               `json:"email"`
			PublicToken string               `json:"public_token"`
			Purpose     models.Purpose       `json:"purpose"`
			MetaData    models.PlaidMetaData `json:"meta_data,omitempty"`
		}
		var input Input
		if err := c.BodyParser(&input); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(err.Error())
		}
		if strings.HasPrefix(input.Email, "public") {
			temp := input.Email
			input.Email = input.PublicToken
			input.PublicToken = temp
			plaidClient.L.Info("INPUT: %+v", input)
		}
		plaidClient.L.Info("METADATA: %+v", input.MetaData)

		user, err := plaidClient.GetUser(input.Email)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to get user for token", "data": err})
		}

		token, err := plaidClient.ExchangePublicToken(plaidClient.C, input.PublicToken)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to exchange for token", "data": err})
		}

		token.User = user
		token.Institution = input.MetaData.Institution.Name
		token.InstitutionID = input.MetaData.Institution.InstitutionId
		token.Purpose = input.Purpose
		plaidClient.L.Info("TOKEN: %+v", token)

		// dbToken, err := plaidClient.GetUserToken(ctx, user)
		// if err == mongo.ErrNoDocuments || c.Method() == http.MethodPost {
		if err = plaidClient.SaveToken(token); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Failure to create access token", "data": err})
		}

		err = GetandSaveAccountDetails(plaidClient, token, c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to get and save account details", "data": err})
		}
		// } else {
		// 	if err = plaidClient.UpdateToken(ctx, dbToken.ID, token.Value, token.ItemId); err != nil {
		// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Failure to update access token", "data": err})
		// 	}
		// }
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "message": "Access token created successfully", "access_token": token.Value, "item_id": token.ItemId, "token": input})
	}
}

func GetandSaveAccountDetails(plaidClient *client.PlaidClient, token *models.Token, c *fiber.Ctx) error {
	accountDetails, err := plaidClient.GetAccountDetails(token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to get account details", "data": err})
	}

	accounts := accountDetails.Accounts
	plaidAccToDBAccId := make(map[string]string)
	transactions := accountDetails.Transactions

	for _, account := range accounts {
		req := &models.CreateAccountRequest{Account: account}
		dbAccount, err := plaidClient.CreateAccount(plaidClient.C, req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to save account account", "data": err})
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
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failure to save account transaction", "data": err})
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
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err)
		}

		type Exist struct {
			Debit  bool `json:"debit"`
			Credit bool `json:"credit"`
		}

		debitAcc, err := plaidClient.IsDebitAccountLinked(user.GetID())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on fetching user's credit accounts", "data": err})
		}
		creditAcc, err := plaidClient.IsCreditAccountLinked(user.GetID())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on fetching user's credit accounts", "data": err})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "data": Exist{debitAcc.Status, creditAcc.Status}})
	}
}
