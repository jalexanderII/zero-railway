package router

import (
	"github.com/go-redis/cache/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/handlers"
	"github.com/jalexanderII/zero-railway/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

func GetUserFromClerkId(UserDb *mongo.Collection, rcache *cache.Cache) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		clerkId := c.Get("Clerk")

		var user models.User

		err := rcache.Get(ctx, clerkId, &user)
		if err != nil && err != cache.ErrCacheMiss {
			return handlers.FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed get user from cache", err.Error())
		}

		if err == cache.ErrCacheMiss && clerkId != "" {
			filter := bson.M{"clerk_id": clerkId}
			err = UserDb.FindOne(ctx, filter).Decode(&user)
			if err != nil {
				l.Errorf("failed to get a user: %s", clerkId)
				return handlers.FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed getting users from headers", err.Error())
			}

			if err = rcache.Set(&cache.Item{
				Ctx:   ctx,
				Key:   clerkId,
				Value: &user,
				TTL:   24 * time.Hour,
			}); err != nil {
				return handlers.FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "failed set user in cache", err.Error())
			}
		}

		return c.Next()
	}
}
