package middleware

import (
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/fiber-gateway/schemas/response"
	"github.com/fiber-gateway/utils"
	"github.com/gofiber/fiber/v2"
)

// JWTProtected is a middleware that validates incoming JWT Access Tokens.
func JWTProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {

		// 1. Get the Authorization header or Cookie
		var tokenStr string
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = authHeader[7:]
		} else if cookie := c.Cookies("access_token"); cookie != "" {
			tokenStr = cookie
		}

		if tokenStr == "" {
			return response.SendError(c, fiber.StatusUnauthorized, "Missing Bearer token or access_token cookie")
		}

		claims, err := utils.ParseToken(tokenStr)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				return response.SendError(c, fiber.StatusUnauthorized, "Token is expired")
			}
			return response.SendError(c, fiber.StatusUnauthorized, "Token is invalid")
		}

		// Security: Only allow tokens specifically issued for Access
		if claims.Type != utils.TokenTypeAccess {
			return response.SendError(c, fiber.StatusUnauthorized, "Token is not a valid Access Token")
		}

		// 4. Attach the UserID and Role to the Fiber Context so subsequent handlers can access it!
		c.Locals("user_id", claims.UserID)
		c.Locals("role", claims.Role)

		// 5. Continue to the next handler/route
		return c.Next()
	}
}
