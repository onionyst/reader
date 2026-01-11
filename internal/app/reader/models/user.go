package models

import (
	"errors"

	"gorm.io/gorm"
)

// User user
type User struct {
	ID int64 `gorm:"primaryKey"`

	Email    string `gorm:"not null;uniqueIndex:uidx_users_email"`
	Password string `gorm:"not null"` // hashed
}

// AddUser adds user for email and hashed password
func AddUser(email, password string) (int64, error) {
	user := &User{
		Email:    email,
		Password: password,
	}
	if res := db.Create(&user); res.Error != nil {
		return 0, res.Error
	}

	return user.ID, nil
}

// GetUser gets user with email
func GetUser(email string) (*User, error) {
	var user *User
	if res := db.Where(&User{Email: email}).First(&user); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, res.Error
	}

	return user, nil
}
