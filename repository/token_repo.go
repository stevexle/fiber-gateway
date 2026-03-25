package repository

import (
	"github.com/fiber-gateway/database"
	"github.com/fiber-gateway/models"
)

func SaveRefreshToken(token models.RefreshToken) error {
	return database.DB.Create(&token).Error
}

func FindAvailableRefreshToken(tokenStr string) (*models.RefreshToken, error) {

	var token models.RefreshToken

	err := database.DB.
		Where("token = ? AND revoked = ?", tokenStr, false).
		First(&token).Error

	return &token, err
}

func RevokeToken(tokenStr string) error {
	return database.DB.Model(&models.RefreshToken{}).
		Where("token = ?", tokenStr).
		Update("revoked", true).Error
}
