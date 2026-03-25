package handler

import (
	"encoding/base64"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/fiber-gateway/models"
	"github.com/fiber-gateway/schemas/response"
	"github.com/fiber-gateway/repository"
	"github.com/fiber-gateway/utils"
	"github.com/gofiber/fiber/v2"
)

func Login(c *fiber.Ctx) error {

	authHeader := c.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Basic ") {
		return response.SendError(c, fiber.StatusUnauthorized, "Missing or invalid Basic Authentication header")
	}

	// Decode the base64 string
	payload, err := base64.StdEncoding.DecodeString(authHeader[6:])
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, "Malformed Basic authentication string")
	}

	// Basic auth payload is "username:password"
	parts := strings.SplitN(string(payload), ":", 2)
	if len(parts) != 2 {
		return response.SendError(c, fiber.StatusBadRequest, "Invalid Basic authentication payload format")
	}

	username := parts[0]
	password := parts[1]

	slog.Info("Login attempt", slog.String("username", username))

	user, err := repository.FindUserByUsername(username)
	if err != nil {
		slog.Warn("Login failed: user not found", slog.String("username", username))
		return response.SendError(c, fiber.StatusUnauthorized, "Invalid credentials")
	}

	if user.Locked {
		slog.Warn("Login failed: account is locked", slog.String("username", username))
		return response.SendError(c, fiber.StatusUnauthorized, "Account is locked due to too many failed login attempts")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		user.Visit++
		if user.Visit >= 3 {
			user.Locked = true
		}
		if repoErr := repository.UpdateUser(user); repoErr != nil {
			slog.Error("Failed to update user login attempts", slog.String("error", repoErr.Error()))
		}

		if user.Locked {
			slog.Warn("Account locked due to too many failed attempts", slog.String("username", username))
			return response.SendError(c, fiber.StatusUnauthorized, "Account has been locked")
		}

		slog.Warn("Login failed: invalid password", slog.String("username", username))
		return response.SendError(c, fiber.StatusUnauthorized, "Invalid credentials")
	}

	if user.Visit > 0 {
		user.Visit = 0
		if repoErr := repository.UpdateUser(user); repoErr != nil {
			slog.Error("Failed to reset user login attempts", slog.String("error", repoErr.Error()))
		}
	}

	accessToken, _ := utils.GenerateAccessToken(user.ID, user.Role)
	refreshToken, _ := utils.GenerateRefreshToken(user.ID)

	if err := repository.SaveRefreshToken(models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(utils.GetRefreshExpDays()),
	}); err != nil {
		slog.Error("Failed to persist refresh token", slog.String("error", err.Error()))
		return response.SendError(c, fiber.StatusInternalServerError, "Internal authentication error")
	}

	slog.Info("Login successful", slog.String("username", username), slog.Any("user_id", user.ID))

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func Refresh(c *fiber.Ctx) error {

	authHeader := c.Get("Authorization")

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return response.SendError(c, fiber.StatusUnauthorized, "Missing or invalid Bearer token")
	}

	// Extract the token without the "Bearer " prefix (7 characters)
	refreshTokenStr := authHeader[7:]

	token, err := repository.FindAvailableRefreshToken(refreshTokenStr)
	if err != nil {
		slog.Warn("Refresh token failed: invalid or expired token")
		return response.SendError(c, fiber.StatusUnauthorized, "Invalid refresh token")
	}

	slog.Info("Token refreshed successfully", slog.Any("user_id", token.UserID))
	accessToken, _ := utils.GenerateAccessToken(token.UserID, models.RoleUser)

	return c.JSON(fiber.Map{
		"access_token": accessToken,
	})
}

func Logout(c *fiber.Ctx) error {

	authHeader := c.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return response.SendError(c, fiber.StatusUnauthorized, "Missing or invalid Bearer token")
	}

	refreshTokenStr := authHeader[7:]

	err := repository.RevokeToken(refreshTokenStr)
	if err != nil {
		slog.Error("Failed to revoke token during logout", slog.String("error", err.Error()))
		return response.SendError(c, fiber.StatusInternalServerError, "Failed to revoke token")
	}

	slog.Info("Logout successful")
	return c.SendStatus(fiber.StatusNoContent)
}
