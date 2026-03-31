package handler

import (
	"github.com/fiber-gateway/middleware"
	"github.com/fiber-gateway/models"
	"github.com/fiber-gateway/repository"
	"github.com/fiber-gateway/schemas/response"
	"github.com/gofiber/fiber/v2"
)

func RegisterClient(c *fiber.Ctx) error {
	type RegisterClientRequest struct {
		ClientID            string `json:"client_id"`
		Name                string `json:"name"`
		RealmID             string `json:"realm_id"`
		Description         string `json:"description"`
		IsConfidential       bool   `json:"is_confidential"`
		// ClientType: "web" | "mobile" | "service"
		ClientType          string `json:"client_type"`
		IconURL             string `json:"icon_url"`
		HomePageURL         string `json:"home_page_url"`
		PrivacyPolicyURL    string `json:"privacy_policy_url"`
		SignInRedirectURIs  string `json:"sign_in_redirect_uris"`
		SignOutRedirectURIs string `json:"sign_out_redirect_uris"`
		WebOrigins          string `json:"web_origins"`
	}

	var req RegisterClientRequest
	if err := c.BodyParser(&req); err != nil {
		return response.SendError(c, 400, "Invalid request payload")
	}

	if req.ClientID == "" {
		return response.SendError(c, 400, "client_id is mandatory")
	}

	if req.Name == "" {
		return response.SendError(c, 400, "application name is required")
	}

	if req.SignInRedirectURIs == "" {
		return response.SendError(c, 400, "at least one redirect_uri is required")
	}

	clientType := req.ClientType
	if clientType == "" {
		clientType = "web" // default
	}

	client := models.Client{
		ClientID:            req.ClientID,
		Name:                req.Name,
		RealmID:             req.RealmID,
		Description:         req.Description,
		IsConfidential:      req.IsConfidential,
		ClientType:          clientType,
		IconURL:             req.IconURL,
		HomePageURL:         req.HomePageURL,
		PrivacyPolicyURL:    req.PrivacyPolicyURL,
		SignInRedirectURIs:  req.SignInRedirectURIs,
		SignOutRedirectURIs: req.SignOutRedirectURIs,
		WebOrigins:          req.WebOrigins,
	}

	var rawSecret string
	if req.IsConfidential {
		rawSecret, _ = repository.GenerateClientSecret()
		client.ClientSecret = rawSecret
	}

	if err := repository.CreateClient(&client); err != nil {
		return response.SendError(c, 500, "Failed to persist application identity")
	}

	middleware.InvalidateOriginCache(client.ClientID)

	return c.Status(201).JSON(fiber.Map{
		"message":       "Identity successfully registered",
		"client_id":      client.ClientID,
		"client_secret":  rawSecret,
		"is_confidential": client.IsConfidential,
	})
}
