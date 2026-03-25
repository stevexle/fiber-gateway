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

		// 1. Get the Authorization header
		authHeader := c.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return response.SendError(c, fiber.StatusUnauthorized, "Missing or invalid Bearer token")
		}

		// 2. Extract the token string without "Bearer "
		tokenStr := authHeader[7:]

		// 3. Parse and validate the token via our utils package
		claims, err := utils.ParseToken(tokenStr)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				return response.SendError(c, fiber.StatusUnauthorized, "Token is expired")
			}
			if errors.Is(err, jwt.ErrTokenMalformed) {
				return response.SendError(c, fiber.StatusUnauthorized, "Token is malformed (not properly formatted)")
			}
			if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
				return response.SendError(c, fiber.StatusUnauthorized, "Token signature is invalid (tampering detected)")
			}
			if errors.Is(err, jwt.ErrTokenNotValidYet) {
				return response.SendError(c, fiber.StatusUnauthorized, "Token is not valid yet")
			}
			return response.SendError(c, fiber.StatusUnauthorized, "Token is completely invalid")
		}

		// 4. Attach the UserID and Role to the Fiber Context so subsequent handlers can access it!
		c.Locals("user_id", claims.UserID)
		c.Locals("role", claims.Role)

		// 5. Continue to the next handler/route
		return c.Next()
	}
}
