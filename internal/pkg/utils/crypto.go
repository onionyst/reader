package utils

import (
	"crypto/sha1"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword generates password hash
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// VerifyPassword verifies password with hash
func VerifyPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Sha1 generates sha1 hash for plain string
func Sha1(plain string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(plain)))
}
