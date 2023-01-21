package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/database"
	"github.com/jalexanderII/zero-railway/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

import "go.mongodb.org/mongo-driver/mongo"

type CreateResDTO struct {
	InsertedId primitive.ObjectID `json:"inserted_id" bson:"_id"`
}

type Handler struct {
	Db *mongo.Collection
	L  *logrus.Logger
	C  context.Context
}

func NewHandler(collectionName string, l *logrus.Logger) *Handler {
	return &Handler{Db: database.GetCollection(collectionName), L: l, C: context.Background()}
}

func (h *Handler) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	filter := bson.M{"email": email}
	err := h.Db.FindOne(h.C, filter).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func FiberJsonResponse(c *fiber.Ctx, httpStatus int, status, message string, data any) error {
	return c.Status(httpStatus).JSON(fiber.Map{"status": status, "message": message, "data": data})
}
