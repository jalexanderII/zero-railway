package handlers

import (
	"github.com/gofiber/fiber/v2"
)

func FiberJsonResponse(c *fiber.Ctx, httpStatus int, status, message string, data any) error {
	return c.Status(httpStatus).JSON(fiber.Map{"status": status, "message": message, "data": data})
}
