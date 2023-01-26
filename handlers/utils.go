package handlers

import (
	"context"
	"net/http"
	"os"
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
