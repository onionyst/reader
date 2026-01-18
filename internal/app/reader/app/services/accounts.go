package services

import (
	"context"
	"fmt"
	"os"
	"strings"

	"reader/internal/app/reader/models"
	"reader/internal/pkg/apperror"
	"reader/internal/pkg/utils"
)

const (
	authPrefix            = "GoogleLogin auth="
	errInvalidCredentials = "failed to authorize user credentials"
)

// CheckUserAuth validates the Authorization header (GoogleLogin auth=email/hash).
func (s *Service) CheckUserAuth(ctx context.Context, authHeader string) (*models.User, error) {
	if !strings.HasPrefix(authHeader, authPrefix) {
		return nil, apperror.Unauthorized(errInvalidCredentials).WithCause(fmt.Errorf("missing Authorization header"))
	}

	sID := strings.TrimPrefix(authHeader, authPrefix)
	email, token, found := strings.Cut(sID, "/")
	if !found {
		return nil, apperror.Unauthorized(errInvalidCredentials).WithCause(fmt.Errorf("missing token"))
	}

	user, userFound, err := s.repo.GetUser(ctx, email)
	if err != nil {
		return nil, apperror.Unauthorized(errInvalidCredentials).WithCause(fmt.Errorf("get user %s: %w", email, err))
	}
	if !userFound {
		return nil, apperror.Unauthorized(errInvalidCredentials).WithCause(fmt.Errorf("user %s not found", email))
	}

	if token != s.generateAuthHash(user) {
		return nil, apperror.Unauthorized(errInvalidCredentials).WithCause(fmt.Errorf("invalid token"))
	}

	return user, nil
}

func (s *Service) ClientLogin(ctx context.Context, email, password string) (string, error) {
	user, found, err := s.repo.GetUser(ctx, email)
	if err != nil {
		return "", apperror.Unauthorized(errInvalidCredentials).WithCause(fmt.Errorf("get user %s: %w", email, err))
	}
	if !found {
		return "", apperror.Unauthorized(errInvalidCredentials).WithCause(fmt.Errorf("user %s not found", email))
	}

	ok, err := utils.VerifyPassword(password, user.Password)
	if err != nil {
		return "", apperror.InternalServerError(fmt.Errorf("verify password: %w", err))
	}
	if !ok {
		return "", apperror.Unauthorized(errInvalidCredentials).WithCause(fmt.Errorf("invalid password"))
	}

	sid := fmt.Sprintf("%s/%s", user.Email, s.generateAuthHash(user))
	return fmt.Sprintf("SID=%s\nLSID=null\nAuth=%s\n", sid, sid), nil
}

func (s *Service) GenerateToken(user *models.User) string {
	salt := os.Getenv("APP_SALT")
	hash := utils.Sha1(fmt.Sprintf("%s%d%s", salt, user.ID, user.Password))
	return utils.PadString(hash, 'Z', 57, false)
}

func (s *Service) generateAuthHash(user *models.User) string {
	salt := os.Getenv("APP_SALT")
	return utils.Sha1(fmt.Sprintf("%s%s%s", salt, user.Email, user.Password))
}
