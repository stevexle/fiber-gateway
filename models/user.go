package models

import "time"

type Role string

const (
	RoleUser    Role = "USER"
	RoleAdmin   Role = "ADMIN"
	RoleService Role = "SERVICE"
)

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"uniqueIndex" json:"username"`
	Password  string         `json:"-"`
	Role      Role           `gorm:"default:'USER'" json:"role"`
	Visit     int            `gorm:"default:0" json:"visit"`
	Locked    bool           `gorm:"default:false" json:"locked"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID" json:"refresh_tokens,omitempty"`
}
