package handlers

import (
	"context"
	"fmt"
	"github.com/go-redis/cache/v8"
	client "github.com/jalexanderII/zero-railway/app/clients"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/database"
	"github.com/jalexanderII/zero-railway/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

import "go.mongodb.org/mongo-driver/mongo"

type DBInsertResponse struct {
	InsertedId primitive.ObjectID `json:"inserted_id" bson:"_id"`
}

type Handler struct {
	Db     *mongo.Collection
	UserDb *mongo.Collection
	P      *client.PlaidClient
	L      *logrus.Logger
	C      context.Context
	H      *http.Client
}

func NewHandler(collectionName string, l *logrus.Logger, p *client.PlaidClient) *Handler {
	return &Handler{
		Db:     database.GetCollection(collectionName),
		UserDb: database.GetCollection(os.Getenv("USER_COLLECTION")),
		P:      p,
		L:      l,
		C:      context.Background(),
		H:      &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *Handler) GetUserByEmail(email string, rcache *cache.Cache) (*models.User, error) {
	var cachedUser models.User
	err := rcache.Get(h.C, email, &cachedUser)
	if err == cache.ErrCacheMiss {
		var user models.User
		filter := bson.M{"email": email}
		err := h.UserDb.FindOne(h.C, filter).Decode(&user)
		if err != nil {
			h.L.Error("[UserDB] Error getting user", "error", err)
			return nil, err
		}

		if err := rcache.Set(&cache.Item{
			Ctx:   h.C,
			Key:   email,
			Value: &user,
			TTL:   24 * time.Hour,
		}); err != nil {
			return nil, err
		}

		return &user, nil
	} else if err != nil {
		return nil, err
	}

	return &cachedUser, nil
}

func (h *Handler) GetUserByID(userId string) (*models.User, error) {
	Id, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, err
	}
	var user models.User
	filter := bson.M{"_id": Id}
	if err = h.UserDb.FindOne(h.C, filter).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func FiberJsonResponse(c *fiber.Ctx, httpStatus int, status, message string, data any) error {
	return c.Status(httpStatus).JSON(fiber.Map{"status": status, "message": message, "data": data})
}

// CreateCookie makes a valid httponly cookie
func CreateCookie(c *fiber.Ctx, name, value string) {
	cookie := &fiber.Cookie{
		Name:     name,
		Value:    value,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
	}

	// Set cookie
	c.Cookie(cookie)
}

// DeleteCookie removes existing cookie
func DeleteCookie(c *fiber.Ctx, name string) {
	c.Cookie(&fiber.Cookie{
		Name: name,
		// Set expiry date to the past
		Expires:  time.Now().Add(-(time.Hour * 2)),
		HTTPOnly: true,
	})
}

// GetPlaidErrorCode will get the error code from the error message and return it as a string
func GetPlaidErrorCode(err error) string {
	errorMessage := err.Error()

	// first get the index of the substring code
	start := strings.Index(errorMessage, ", code: ") + 8

	// get the end by creating a substring and getting the index of the first comma
	end := strings.Index(errorMessage[start:], ", ") + start

	// return the substring with the window of indexes
	return errorMessage[start:end]
}

func FormatPhoneNumber(pn string) string {
	// if the first char isn't a plus, add it
	if pn[0:1] != "+" {
		return fmt.Sprintf("+%s", pn)
	}
	return pn
}

func FetchDataAndCache(userID primitive.ObjectID, plaidClient *client.PlaidClient, rcache *cache.Cache, reset bool) (*models.AccountDetailsResponse, error) {
	var cachedAccountDetails models.AccountDetailsResponse
	if reset {
		err := rcache.Delete(plaidClient.C, userID.Hex())
		if err != nil {
			return nil, err
		}
	}
	err := rcache.Get(plaidClient.C, userID.Hex(), &cachedAccountDetails)
	if err == cache.ErrCacheMiss || reset {
		tokens, err := plaidClient.GetTokens(userID)
		if err != nil {
			return nil, err
		}

		var accounts []*models.Account
		var transactions []*models.Transaction
		for _, token := range *tokens {
			accountDetails, err := plaidClient.GetAccountDetails(&token)
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, accountDetails.Accounts...)
			transactions = append(transactions, accountDetails.Transactions...)
		}
		consolidatedAccountDetails := models.AccountDetailsResponse{
			Accounts:     accounts,
			Transactions: transactions,
		}

		if err := rcache.Set(&cache.Item{
			Ctx:   plaidClient.C,
			Key:   userID.Hex(),
			Value: &consolidatedAccountDetails,
			TTL:   24 * time.Hour,
		}); err != nil {
			return nil, err
		}

		return &consolidatedAccountDetails, nil
	} else if err != nil {
		return nil, err
	}

	return &cachedAccountDetails, nil
}

func FetchAccountDetails(userID primitive.ObjectID, plaidClient *client.PlaidClient, rcache *cache.Cache) ([]*models.Account, error) {
	AccountDetails, err := FetchDataAndCache(userID, plaidClient, rcache, false)
	if err != nil {
		return nil, err
	}
	return AccountDetails.Accounts, nil
}

func FetchTransactionDetails(userID primitive.ObjectID, plaidClient *client.PlaidClient, rcache *cache.Cache) ([]*models.Transaction, error) {
	AccountDetails, err := FetchDataAndCache(userID, plaidClient, rcache, false)
	if err != nil {
		return nil, err
	}
	return AccountDetails.Transactions, nil
}
