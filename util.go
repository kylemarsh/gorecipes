package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rivo/uniseg"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrIconValidation = errors.New("icon validation failed")
	ErrLabelConflict  = errors.New("label name conflict")
	ErrTypeValidation = errors.New("type validation failed")
)

type CustomClaims struct {
	UserID  int  `json:"user_id"`
	IsAdmin bool `json:"is_admin"`
	jwt.RegisteredClaims
}

func readConfiguration(c *configuration, configFilename string) error {
	file, err := os.Open(configFilename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&c)
}

func jwtGenerate(userID int, isAdmin bool) (string, error) {
	// 1 month expiration. TODO Decide on final scheme?
	claims := &CustomClaims{
		UserID:  userID,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 30)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(conf.JwtSecret))

	if err != nil {
		return "", err
	}

	if conf.Debug {
		fmt.Println("Generated Token:")
		fmt.Println(token)
		fmt.Println(tokenStr)
	}

	return tokenStr, nil
}

func jwtExtractClaims(tokenString string) (*CustomClaims, error) {
	if tokenString == "" {
		return nil, errors.New("missing auth token")
	}

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.JwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

func hashPassword(password string) (string, error) {
	var pwBytes = []byte(password)
	hashedBytes, err := bcrypt.GenerateFromPassword(pwBytes, bcrypt.MinCost)
	return string(hashedBytes), err
}

func validateIcon(icon string) error {
	if icon == "" {
		return nil // Empty is valid
	}

	count := uniseg.GraphemeClusterCount(icon)
	if count != 1 {
		return fmt.Errorf("icon must be exactly 1 character, got %d: %w", count, ErrIconValidation)
	}
	return nil
}

func validateType(labelType string) error {
	if labelType == "" {
		return nil // Empty is valid
	}

	if len(labelType) > 20 {
		return fmt.Errorf("type must be 20 characters or less, got %d: %w", len(labelType), ErrTypeValidation)
	}
	return nil
}
