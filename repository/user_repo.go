package repository

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/fiber-gateway/database"
	"github.com/fiber-gateway/models"
)

func FindUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := database.DB.
		Where("username = ?", username).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func UpdateUser(user *models.User) error {
	return database.DB.Save(user).Error
}

func CreateUser(username, password string, role models.Role) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Username: username,
		Password: string(hashedPassword),
		Role:     role,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
