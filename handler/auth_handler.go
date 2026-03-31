package handler

import (
	"log/slog"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/fiber-gateway/config"
	"github.com/fiber-gateway/models"
	"github.com/fiber-gateway/repository"
	"github.com/fiber-gateway/schemas/response"
	"github.com/fiber-gateway/utils"
	"github.com/gofiber/fiber/v2"
)

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	ClientID string `json:"client_id"`
}

type AuthorizeRequest struct {
	ClientID            string `json:"client_id" query:"client_id"`
	RedirectURI         string `json:"redirect_uri" query:"redirect_uri"`
	State               string `json:"state" query:"state"`
	CodeChallenge       string `json:"code_challenge" query:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method" query:"code_challenge_method"`
}

type TokenExchangeRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
}

// getToken extracts the token from Authorization header or falls back to a specific cookie
func getToken(c *fiber.Ctx, cookieName string) string {
	authHeader := c.Get("Authorization")
	if after, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
		return after
	}
	return c.Cookies(cookieName)
}

func RegisterUser(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return response.SendError(c, 400, "Invalid payload")
	}
	_, err := repository.CreateUser(user.Username, user.Password, models.RoleUser)
	if err != nil {
		return response.SendError(c, 500, "Registration failed")
	}
	return c.Status(201).JSON(fiber.Map{"message": "User registered"})
}

func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.SendError(c, 400, "Invalid payload")
	}
	user, err := repository.FindUserByUsername(req.Username)
	if err != nil {
		return response.SendError(c, fiber.StatusUnauthorized, "Invalid credentials")
	}

	if user.Locked {
		slog.Warn("Attempt to login to locked account", slog.String("username", req.Username))
		return response.SendError(c, fiber.StatusUnauthorized, "Account has been locked")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		user.Visit++
		if user.Visit >= 3 {
			user.Locked = true
		}
		if repoErr := repository.UpdateUser(user); repoErr != nil {
			slog.Error("Failed to update user login attempts", slog.String("error", repoErr.Error()))
		}

		if user.Locked {
			slog.Warn("Account locked due to too many failed attempts", slog.String("username", req.Username))
			return response.SendError(c, fiber.StatusUnauthorized, "Account has been locked")
		}

		slog.Warn("Login failed: invalid password", slog.String("username", req.Username))
		return response.SendError(c, fiber.StatusUnauthorized, "Invalid credentials")
	}

	if user.Visit > 0 {
		user.Visit = 0
		if repoErr := repository.UpdateUser(user); repoErr != nil {
			slog.Error("Failed to reset user login attempts", slog.String("error", repoErr.Error()))
		}
	}
	// Determine IP binding based on client_type
	// Mobile clients change IPs frequently (WiFi <-> cellular), so we skip IP binding.
	ipBinding := c.IP()
	if req.ClientID != "" {
		if client, err := repository.FindClientByID(req.ClientID); err == nil && client.ClientType == "mobile" {
			ipBinding = ""
		}
	}

	token, _ := utils.GenerateSessionToken(user.ID, user.Role, ipBinding)
	c.Cookie(&fiber.Cookie{
		Name:     "session_id",
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   config.AppConfig.Environment == "production",
		SameSite: "Lax",
		Expires:  time.Now().Add(utils.GetSSOSessionExpDays()),
	})

	return c.JSON(fiber.Map{"message": "Proceed to authorize"})
}

func Authorize(c *fiber.Ctx) error {
	var req AuthorizeRequest
	
	// Support both POST (SPA) and GET (Standard OAuth2 Redirect)
	if c.Method() == "GET" {
		c.QueryParser(&req)
	} else {
		c.BodyParser(&req)
	}

	token := getToken(c, "session_id")
	claims, err := utils.ParseToken(token)
	if err != nil || claims.Type != utils.TokenTypeAuthSession {
		return response.SendError(c, 401, "Invalid or expired SSO session")
	}
	// IP binding: only enforce if the session was issued with an IP (web clients)
	if claims.SourceIP != "" && claims.SourceIP != c.IP() {
		slog.Warn("Session Hijacking Detected", slog.String("token_ip", claims.SourceIP), slog.String("request_ip", c.IP()))
		return response.SendError(c, 401, "Session origin IP mismatch")
	}

	client, err := repository.FindClientByID(req.ClientID)
	if err != nil || !isRedirectURIValid(req.RedirectURI, client.SignInRedirectURIs) {
		return response.SendError(c, 403, "Invalid client or redirect")
	}

	// For mobile clients: IP binding is intentionally skipped (network changes frequently)
	// For web clients: enforce strict SourceIP match
	if client.ClientType != "mobile" && claims.SourceIP != "" && claims.SourceIP != c.IP() {
		slog.Warn("Session Hijacking Detected", slog.String("token_ip", claims.SourceIP), slog.String("request_ip", c.IP()))
		return response.SendError(c, 401, "Session origin IP mismatch")
	}

	code, _ := utils.GenerateRandomString(32)
	repository.CreateAuthorizeCode(&models.AuthorizeCode{
		Code: code, CodeChallenge: req.CodeChallenge, CodeChallengeMethod: req.CodeChallengeMethod,
		UserID: claims.UserID, UserRole: claims.Role, ClientID: req.ClientID,
		RedirectURI: req.RedirectURI, State: req.State, ExpiresAt: time.Now().Add(utils.GetAuthCodeExpMinutes()),
	})

	u, _ := url.Parse(req.RedirectURI)
	q := u.Query()
	q.Set("code", code)
	if req.State != "" { q.Set("state", req.State) }
	u.RawQuery = q.Encode()

	// Handle standard OAuth2 GET flow with 302 Redirect for cross-domain SSO
	if c.Method() == "GET" {
		return c.Redirect(u.String(), fiber.StatusFound)
	}

	// Fallback for current internal SPA
	return c.JSON(fiber.Map{"redirect_uri": u.String()})
}

func ExchangeToken(c *fiber.Ctx) error {
	var req TokenExchangeRequest
	if err := c.BodyParser(&req); err != nil {
		return response.SendError(c, 400, "Malformed exchange request")
	}

	if req.GrantType == "client_credentials" {
		client, err := repository.FindClientByID(req.ClientID)
		if err != nil {
			return response.SendError(c, 401, "Invalid client_id")
		}
		if !client.IsConfidential {
			return response.SendError(c, 403, "M2M flow is only allowed for confidential clients")
		}

		if !repository.VerifyClientSecret(req.ClientSecret, client.ClientSecret) {
			return response.SendError(c, 401, "Invalid client_secret")
		}

		token, _ := utils.GenerateClientToken(client.ClientID)
		return c.JSON(fiber.Map{
			"access_token": token,
			"token_type":   "Bearer",
			"expires_in":   int(utils.GetAccessExpMinutes().Seconds()),
		})
	}

	authCode, err := repository.GetAuthorizeCode(req.Code)
	if err != nil || authCode.ClientID != req.ClientID {
		return response.SendError(c, 401, "Invalid code or client")
	}
	
	if authCode.ExpiresAt.Before(time.Now()) {
		repository.MarkCodeAsUsed(req.Code)
		return response.SendError(c, 401, "Authorization code has expired")
	}

	client, _ := repository.FindClientByID(req.ClientID)
	if client.IsConfidential {
		if !repository.VerifyClientSecret(req.ClientSecret, client.ClientSecret) {
			return response.SendError(c, 401, "Invalid client_secret for confidential exchange")
		}
	} else if req.CodeVerifier == "" {
		return response.SendError(c, 401, "Security mismatch: Code Verifier is required for public clients")
	}

	if utils.VerifyPKCE(req.CodeVerifier, authCode.CodeChallenge, authCode.CodeChallengeMethod) != nil {
		return response.SendError(c, 401, "Invalid PKCE verifier")
	}

	repository.MarkCodeAsUsed(req.Code)
	accessToken, _ := utils.GenerateAccessToken(authCode.UserID, authCode.UserRole)

	isMobile := client.ClientType == "mobile"
	isSecure  := config.AppConfig.Environment == "production"

	// Mobile: no IP binding (clients change network frequently)
	ipBinding := c.IP()
	if isMobile {
		ipBinding = ""
	}

	refreshToken, _ := utils.GenerateRefreshToken(authCode.UserID, ipBinding)
	repository.SaveRefreshToken(models.RefreshToken{
		UserID: authCode.UserID, Token: refreshToken, ExpiresAt: time.Now().Add(utils.GetRefreshExpDays()),
	})

	// Mobile: return tokens in response body for Keychain/Keystore storage
	if isMobile {
		return c.JSON(fiber.Map{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"expires_in":    int(utils.GetAccessExpMinutes().Seconds()),
		})
	}

	// Web: set HttpOnly cookies — JS cannot read them
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HTTPOnly: true,
		Secure:   isSecure,
		SameSite: "Lax",
		Expires:  time.Now().Add(utils.GetAccessExpMinutes()),
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HTTPOnly: true,
		Secure:   isSecure,
		SameSite: "Lax",
		Expires:  time.Now().Add(utils.GetRefreshExpDays()),
	})

	return c.JSON(fiber.Map{
		"expires_in": int(utils.GetAccessExpMinutes().Seconds()),
		"token_type": "Bearer",
		"message":    "Tokens securely stored via HttpOnly cookies",
	})
}

func Refresh(c *fiber.Ctx) error {
	refreshTokenStr := getToken(c, "refresh_token")
	if refreshTokenStr == "" {
		return response.SendError(c, 401, "Missing refresh_token in Authorization header or cookie")
	}

	token, err := repository.FindAvailableRefreshToken(refreshTokenStr)
	if err != nil || token.ExpiresAt.Before(time.Now()) {
		return response.SendError(c, 401, "Invalid or expired refresh token")
	}

	user, _ := repository.FindUserByID(token.UserID)
	newAccess, _ := utils.GenerateAccessToken(user.ID, user.Role)

	// Detect mobile: they send via Authorization header (not cookie)
	isMobileRefresh := c.Get("Authorization") != ""

	if isMobileRefresh {
		// Mobile: return new access_token in JSON body for Keychain/Keystore update
		return c.JSON(fiber.Map{
			"access_token": newAccess,
			"token_type":   "Bearer",
			"expires_in":   int(utils.GetAccessExpMinutes().Seconds()),
		})
	}

	// Web: silently rotate via HttpOnly cookie
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    newAccess,
		Path:     "/",
		HTTPOnly: true,
		Secure:   config.AppConfig.Environment == "production",
		SameSite: "Lax",
		Expires:  time.Now().Add(utils.GetAccessExpMinutes()),
	})

	return c.JSON(fiber.Map{
		"expires_in": int(utils.GetAccessExpMinutes().Seconds()),
		"token_type": "Bearer",
		"message":    "Silent token refresh successful",
	})
}

func Logout(c *fiber.Ctx) error {
	refreshTokenStr := getToken(c, "refresh_token")
	if refreshTokenStr != "" {
		repository.RevokeToken(refreshTokenStr)
	}

	// Wipe cookies securely
	c.Cookie(&fiber.Cookie{Name: "session_id", Value: "", Path: "/", Expires: time.Now().Add(-time.Hour)})
	c.Cookie(&fiber.Cookie{Name: "access_token", Value: "", Path: "/", Expires: time.Now().Add(-time.Hour)})
	c.Cookie(&fiber.Cookie{Name: "refresh_token", Value: "", Path: "/", Expires: time.Now().Add(-time.Hour)})

	return c.JSON(fiber.Map{"message": "Logged out and token revoked"})
}

func isRedirectURIValid(input, allowed string) bool {
	for _, item := range strings.Split(allowed, ",") {
		if strings.TrimSpace(item) == input { return true }
	}
	return false
}
