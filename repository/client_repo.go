package repository

import (
	"golang.org/x/crypto/bcrypt"
	"github.com/fiber-gateway/database"
	"github.com/fiber-gateway/models"
	"github.com/fiber-gateway/utils"
)

func FindClientByID(clientID string) (*models.Client, error) {
	var client models.Client
	err := database.DB.Where("client_id = ?", clientID).First(&client).Error
	return &client, err
}

func CreateClient(client *models.Client) error {
	// Security: Hash the secret before storage if it exists
	if client.ClientSecret != "" {
		hashed, _ := bcrypt.GenerateFromPassword([]byte(client.ClientSecret), 10)
		client.ClientSecret = string(hashed)
	}
	return database.DB.Create(client).Error
}

func VerifyClientSecret(rawSecret, hashedSecret string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedSecret), []byte(rawSecret)) == nil
}

func FindAllByRealmID(realmID string) ([]models.Client, error) {
	var clients []models.Client
	err := database.DB.Where("realm_id = ?", realmID).Find(&clients).Error
	return clients, err
}

func GenerateClientSecret() (string, error) {
	return utils.GenerateRandomString(32)
}
