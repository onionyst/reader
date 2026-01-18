package models

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"reader/internal/pkg/db/postgres"
)

var ErrUsersEmailAlreadyExists = errors.New("email already exists")

const (
	constraintUsersEmail = "uidx_users_email"
)

var usersUniqueConstraintErr = map[string]error{
	constraintUsersEmail: ErrUsersEmailAlreadyExists,
}

type User struct {
	ID int64 `gorm:"primaryKey"`

	Email    string `gorm:"not null;uniqueIndex:uidx_users_email"`
	Password string `gorm:"not null"` // hashed
}

// AddUser creates a user with email and hashed password.
func (r *Repo) AddUser(ctx context.Context, email, password string) (int64, error) {
	user := &User{
		Email:    email,
		Password: password,
	}
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return 0, postgres.MapUniqueConstraint(err, usersUniqueConstraintErr)
	}
	return user.ID, nil
}

// GetUser returns a user by email.
func (r *Repo) GetUser(ctx context.Context, email string) (*User, bool, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("email = ?", email).Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &user, true, nil
}
