package auth

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		return "", errors.New("empty password")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func VerifyPassword(passwordHash string, password string) bool {
	passwordHash = strings.TrimSpace(passwordHash)
	password = strings.TrimSpace(password)
	if passwordHash == "" || password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}
