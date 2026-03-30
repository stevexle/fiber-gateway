package repository

import (
	"time"

	"github.com/fiber-gateway/database"
	"github.com/fiber-gateway/models"
)

func CreateAuthorizeCode(code *models.AuthorizeCode) error {
	return database.DB.Create(code).Error
}

func GetAuthorizeCode(codeStr string) (*models.AuthorizeCode, error) {
	var code models.AuthorizeCode
	err := database.DB.Where("code = ? AND used = ? AND expires_at > ?", codeStr, false, time.Now()).First(&code).Error
	if err != nil {
		return nil, err
	}
	return &code, nil
}

func MarkCodeAsUsed(codeStr string) error {
	return database.DB.Model(&models.AuthorizeCode{}).Where("code = ?", codeStr).Update("used", true).Error
}
