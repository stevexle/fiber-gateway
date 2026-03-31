package utils

import (
	"time"

	"github.com/fiber-gateway/config"
	"github.com/fiber-gateway/models"
	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess      = "access"
	TokenTypeAuthSession = "auth_session"
)

type Claims struct {
	UserID   uint        `json:"user_id,omitempty"`
	Role     models.Role `json:"role"`
	ClientID string      `json:"client_id,omitempty"`
	Type     string      `json:"type"`
	SourceIP string      `json:"src_ip,omitempty"`
	jwt.RegisteredClaims
}

func GetAccessExpMinutes() time.Duration   { return config.AppConfig.JWTAccessExpMinutes }
func GetRefreshExpDays() time.Duration     { return config.AppConfig.JWTRefreshExpDays }
func GetAuthCodeExpMinutes() time.Duration { return config.AppConfig.JWTAuthCodeExpMinutes }

// createToken is a core private helper for DRY
func createToken(claims Claims, duration time.Duration) (string, error) {
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(duration))
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(config.AppConfig.JWTSecret)
}

func GenerateAccessToken(userID uint, role models.Role) (string, error) {
	return createToken(Claims{UserID: userID, Role: role, Type: TokenTypeAccess}, GetAccessExpMinutes())
}

func GenerateClientToken(clientID string) (string, error) {
	return createToken(Claims{ClientID: clientID, Role: models.RoleService, Type: TokenTypeAccess}, GetAccessExpMinutes())
}

func GetSSOSessionExpDays() time.Duration { return config.AppConfig.JWTSsoSessionDays }

// GenerateSessionToken issues a SSO session token.
// ip: pass c.IP() for web clients to enable IP binding; pass "" for mobile clients to disable.
func GenerateSessionToken(userID uint, role models.Role, ip string) (string, error) {
	return createToken(Claims{UserID: userID, Role: role, Type: TokenTypeAuthSession, SourceIP: ip}, GetSSOSessionExpDays())
}

// GenerateRefreshToken issues a refresh token for M2M or Web/Mobile.
// ip: pass c.IP() for web clients; pass "" for mobile clients.
func GenerateRefreshToken(userID uint, ip string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"src_ip":  ip,
		"exp":     time.Now().Add(GetRefreshExpDays()).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(config.AppConfig.JWTSecret)
}

func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		return config.AppConfig.JWTSecret, nil
	})
	if err != nil || !token.Valid { return nil, err }
	return token.Claims.(*Claims), nil
}
