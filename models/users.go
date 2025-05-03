package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	gorm.Model
	Username         string `gorm:"uniqueIndex;not null"`
	Email            string `gorm:"uniqueIndex;not null"`
	Password         string
	IsActive         bool `gorm:"default:false"`
	Token            int  `gorm:"default:100"`
	OAuthProvider    string
	OAuthID          string
	OAuthToken       string
	VerificationCode string
	CodeExpiry       *time.Time
	EmailVerified    bool `gorm:"default:false"`
}

