package middleware

import (
	"slices"

	"github.com/fiber-gateway/models"
	"github.com/gofiber/fiber/v2"
)

func RequireRole(roles ...models.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {

		roleInter := c.Locals("role")
		if roleInter == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		var role models.Role
		switch r := roleInter.(type) {
		case models.Role:
			role = r
		case string:
			role = models.Role(r)
		default:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		if slices.Contains(roles, role) {
			return c.Next()
		}

		return c.Status(403).SendString("Forbidden")
	}
}
