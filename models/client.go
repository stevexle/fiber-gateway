package models

import (
	"gorm.io/gorm"
)

// Client represents a registered application in the OAuth 2.0 / OIDC ecosystem.
// This structure follows the enterprise-grade RegisteredClient configuration from Spring Security.
type Client struct {
	gorm.Model
	// ClientID is the unique identifier for the application (e.g., "my-mobile-app").
	ClientID            string `gorm:"uniqueIndex;not null" json:"client_id"`
	// ClientSecret is a hashed secret for confidential clients. Public clients should leave this empty.
	ClientSecret        string `json:"-"`
	// RealmID allows for multi-tenant support, grouping clients within a shared authentication space.
	RealmID             string `json:"realm_id"`
	// Name is a human-readable name displayed on authorization consent screens.
	Name                string `json:"name"`
	// IsConfidential indicates if the client can securely store secrets (True for Server-side, False for SPAs/Mobile).
	IsConfidential      bool   `json:"is_confidential"`
	// ClientType distinguishes the runtime environment: "web" (SPA/Browser), "mobile" (iOS/Android), "service" (M2M).
	// This drives token delivery strategy: cookies for web, response body for mobile.
	ClientType          string `gorm:"default:'web'" json:"client_type"`
	// IconURL is a link to the application logo.
	IconURL             string `json:"icon_url"`
	// HomePageURL is the main URL of the client application.
	HomePageURL         string `json:"home_page_url"`
	// Description provides brief details about the application's purpose.
	Description         string `json:"description"`
	// PrivacyPolicyURL is the required link to the application's legal privacy terms.
	PrivacyPolicyURL    string `json:"privacy_policy_url"`
	// SignInRedirectURIs is a whitelist of comma-separated URLs the gateway can redirect code/tokens back to.
	SignInRedirectURIs  string `json:"sign_in_redirect_uris"`  
	// SignOutRedirectURIs is a whitelist of comma-separated URLs the gateway can redirect back to after logout.
	SignOutRedirectURIs string `json:"sign_out_redirect_uris"` 
	// WebOrigins (CORS) is a comma-separated list of origins allowed to make cross-site requests to the gateway.
	WebOrigins          string `json:"web_origins"`           
}
