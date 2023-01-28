package handlers

import (
	"context"
	"fmt"
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
	L      *logrus.Logger
	C      context.Context
	H      *http.Client
}

func NewHandler(collectionName string, l *logrus.Logger) *Handler {
	return &Handler{
		Db:     database.GetCollection(collectionName),
		UserDb: database.GetCollection(os.Getenv("USER_COLLECTION")),
		L:      l,
		C:      context.Background(),
		H:      &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *Handler) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	filter := bson.M{"email": email}
	err := h.UserDb.FindOne(h.C, filter).Decode(&user)
	if err != nil {
		h.L.Error("[UserDB] Error getting user", "error", err)
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
