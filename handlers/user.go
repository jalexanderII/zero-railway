package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

// @Summary Create a user.
// @Description create a single user.
// @Tags user
// @Accept json
// @Param user body models.User true "User to create"
// @Produce json
// @Success 200 {object} DBInsertResponse
// @Router /users [post]
func CreateUser(h *Handler) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		nUser := new(models.User)
		if err := c.BodyParser(nUser); err != nil {
			return FiberJsonResponse(c, fiber.StatusBadRequest, "error", "request body malformed", err)
		}

		user, err := h.GetUserByEmail(nUser.Email)
		if user == nil || err != nil {
			// ErrNoDocuments means that the filter did not match any documents in the collection
			if user == nil || err == mongo.ErrNoDocuments {
				nUser.ID = primitive.NewObjectID()
				nUser.CreatedAt = time.Now()
				nUser.UpdatedAt = time.Now()
				res, err := h.Db.InsertOne(h.C, nUser)
				if err != nil {
					return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed to create user", err)
				}
				return FiberJsonResponse(c, fiber.StatusOK, "success", "new user created", res.InsertedID)
			}
			h.L.Error("[UserDB] Error checking if user already exists", "error", err)
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "error checking if user already exists", err)
		}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "users already exists", DBInsertResponse{user.ID})
	}
}

// @Summary Get a single user.
// @Description fetch a single user.
// @Tags users
// @Param email path string true "User email"
// @Produce json
// @Success 200 {object} models.User
// @Router /users/:email [get]
func GetUser(h *Handler) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")
		user, err := h.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err)
		}

		return FiberJsonResponse(c, fiber.StatusOK, "success", "found user", user)
	}
}

// @Summary Get a single user.
// @Description fetch a single user by user id.
// @Tags users
// @Param id path string true "User ID"
// @Produce json
// @Success 200 {object} models.User
// @Router /user/:id [get]
func GetUserByID(h *Handler) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusBadRequest, "error", "invalid user id", err)
		}
		var user models.User
		filter := bson.M{"_id": userId}
		if err = h.Db.FindOne(h.C, filter).Decode(&user); err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err)
		}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "user", user)
	}
}
