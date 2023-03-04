package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
			return FiberJsonResponse(c, fiber.StatusBadRequest, "error", "request body malformed", err.Error())
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
					return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed to create user", err.Error())
				}
				return FiberJsonResponse(c, fiber.StatusOK, "success", "new user created", res.InsertedID)
			}
			h.L.Error("[UserDB] Error checking if user already exists", "error", err.Error())
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "error checking if user already exists", err.Error())
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
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err.Error())
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
			return FiberJsonResponse(c, fiber.StatusBadRequest, "error", "invalid user id", err.Error())
		}
		var user models.User
		filter := bson.M{"_id": userId}
		if err = h.Db.FindOne(h.C, filter).Decode(&user); err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err.Error())
		}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "user", user)
	}
}

type UpdateInput struct {
	PhoneNumber string `json:"phoneNumber" bson:"phone_number"`
}

type UpdateResponse struct {
	ModifiedCount int64 `json:"modified_count"`
}

// @Summary Update a users phone number.
// @Description update a single users phone number.
// @Tags users
// @Accept json
// @Param input body UpdateInput true "Update request"
// @Param email path string true "User Email"
// @Produce json
// @Success 200 {object} UpdateResponse
// @Router /users/:email [put]
func UpdateUserPhone(h *Handler) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")
		user, err := h.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err.Error())
		}

		uUser := new(UpdateInput)
		if err = c.BodyParser(uUser); err != nil {
			return FiberJsonResponse(c, fiber.StatusBadRequest, "error", "request body malformed", err.Error())
		}
		if user.PhoneNumber != uUser.PhoneNumber {
			uUser.PhoneNumber = fmt.Sprintf("+1%s", uUser.PhoneNumber)
			h.L.Info("User phone number updated", "user", user.Email, "phone_number", uUser.PhoneNumber)

			filter := bson.M{"_id": user.GetID()}
			update := bson.M{"$set": uUser}
			res, err := h.Db.UpdateOne(h.C, filter, update)
			if err != nil {
				return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed to update user", err.Error())
			}
			return FiberJsonResponse(c, fiber.StatusOK, "success", "updated user", UpdateResponse{res.ModifiedCount})
		}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "no update needed", UpdateResponse{0})
	}
}

// @Summary Create a user.
// @Description create a single user from clerk webhook.
// @Tags user
// @Accept json
// @Param user body models.ClerkUserEvent true "User to create"
// @Produce json
// @Success 200 {object} DBInsertResponse
// @Router /users/clerk [post]
func CreateUserClerkWebhook(h *Handler) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		nUserWebhook := new(models.ClerkUserEvent)
		if err := c.BodyParser(nUserWebhook); err != nil {
			return FiberJsonResponse(c, fiber.StatusBadRequest, "error", "request body malformed", err.Error())
		}

		uEmail := nUserWebhook.Data.GetEmail()
		user, err := h.GetUserByEmail(uEmail)
		if user == nil || err != nil {
			// ErrNoDocuments means that the filter did not match any documents in the collection
			if user == nil || err == mongo.ErrNoDocuments {
				nUser := nUserWebhook.Data.NewDBUser()
				h.L.Info("[ClerkWebhook] Create User Body", nUser)
				res, err := h.Db.InsertOne(h.C, nUser)
				if err != nil {
					return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed to create user", err.Error())
				}
				h.L.Info("[ClerkWebhook] Create User success")
				return FiberJsonResponse(c, fiber.StatusOK, "success", "new user created", res.InsertedID)
			}
			h.L.Error("[UserDB] Error checking if user already exists", "error", err.Error())
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "error checking if user already exists", err.Error())
		}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "users already exists", DBInsertResponse{user.ID})
	}
}

// // @Summary Create a user.
// // @Description create a single user.
// // @Tags user
// // @Accept json
// // @Param user body models.ClerkUserDeleted true "User to create"
// // @Produce json
// // @Success 200 {object} DBInsertResponse
// // @Router /users [post]
// func DeleteUserClerkWebhook(h *Handler) func(c *fiber.Ctx) error {
// 	return func(c *fiber.Ctx) error {
// 		nUser := new(models.User)
// 		if err := c.BodyParser(nUser); err != nil {
// 			return FiberJsonResponse(c, fiber.StatusBadRequest, "error", "request body malformed", err.Error())
// 		}
//
// 		user, err := h.GetUserByEmail(nUser.Email)
// 		if user == nil || err != nil {
// 			// ErrNoDocuments means that the filter did not match any documents in the collection
// 			if user == nil || err == mongo.ErrNoDocuments {
// 				nUser.ID = primitive.NewObjectID()
// 				nUser.CreatedAt = time.Now()
// 				nUser.UpdatedAt = time.Now()
// 				res, err := h.Db.InsertOne(h.C, nUser)
// 				if err != nil {
// 					return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed to create user", err.Error())
// 				}
// 				return FiberJsonResponse(c, fiber.StatusOK, "success", "new user created", res.InsertedID)
// 			}
// 			h.L.Error("[UserDB] Error checking if user already exists", "error", err.Error())
// 			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "error checking if user already exists", err.Error())
// 		}
// 		return FiberJsonResponse(c, fiber.StatusOK, "success", "users already exists", DBInsertResponse{user.ID})
// 	}
// }
//
// // @Summary Create a user.
// // @Description create a single user.
// // @Tags user
// // @Accept json
// // @Param user body models.ClerkUserEvent true "User to create"
// // @Produce json
// // @Success 200 {object} DBInsertResponse
// // @Router /users [post]
// func UpdateUserClerkWebhook(h *Handler) func(c *fiber.Ctx) error {
// 	return func(c *fiber.Ctx) error {
// 		nUser := new(models.User)
// 		if err := c.BodyParser(nUser); err != nil {
// 			return FiberJsonResponse(c, fiber.StatusBadRequest, "error", "request body malformed", err.Error())
// 		}
//
// 		user, err := h.GetUserByEmail(nUser.Email)
// 		if user == nil || err != nil {
// 			// ErrNoDocuments means that the filter did not match any documents in the collection
// 			if user == nil || err == mongo.ErrNoDocuments {
// 				nUser.ID = primitive.NewObjectID()
// 				nUser.CreatedAt = time.Now()
// 				nUser.UpdatedAt = time.Now()
// 				res, err := h.Db.InsertOne(h.C, nUser)
// 				if err != nil {
// 					return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed to create user", err.Error())
// 				}
// 				return FiberJsonResponse(c, fiber.StatusOK, "success", "new user created", res.InsertedID)
// 			}
// 			h.L.Error("[UserDB] Error checking if user already exists", "error", err.Error())
// 			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "error checking if user already exists", err.Error())
// 		}
// 		return FiberJsonResponse(c, fiber.StatusOK, "success", "users already exists", DBInsertResponse{user.ID})
// 	}
// }

func CleanUp(h *Handler) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var results []models.User
		cursor, err := h.UserDb.Find(h.C, bson.D{})
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err.Error())
		}
		if err = cursor.All(h.C, &results); err != nil {
			h.L.Error("[DB] Error getting all users", "error", err.Error())
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err.Error())
		}
		var count = 0
		for _, user := range results {
			if user.PhoneNumber == "+1undefined" {
				test := c.Params("test")
				if test == "false" {
					filter := bson.D{{Key: "_id", Value: user.ID}}
					_, err = h.UserDb.DeleteOne(h.C, filter)
					if err == nil {
						count++
					} else {
						h.L.Error("[DB] Error deleting user", "error", err.Error())
					}
				} else {
					h.L.Info("[DB] Test mode, not deleting user", "user", user.PhoneNumber)
				}
			}
		}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "done deleting users", count)
	}
}
