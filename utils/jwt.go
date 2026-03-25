package utils

import (
	"time"

	"github.com/fiber-gateway/config"
	"github.com/fiber-gateway/models"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint        `json:"user_id"`
	Role   models.Role `json:"role"`
	jwt.RegisteredClaims
}

func GetAccessExpMinutes() time.Duration {
	return config.AppConfig.JWTAccessExpMinutes
}

func GetRefreshExpDays() time.Duration {
	return config.AppConfig.JWTRefreshExpDays
}

func getJWTSecret() []byte {
	return config.AppConfig.JWTSecret
}

func GenerateAccessToken(userID uint, role models.Role) (string, error) {

	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(GetAccessExpMinutes())),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(getJWTSecret())
}

func GenerateRefreshToken(userID uint) (string, error) {

	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(GetRefreshExpDays()).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(getJWTSecret())
}

func ParseToken(tokenStr string) (*Claims, error) {

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, err
	}

	return claims, nil
}
