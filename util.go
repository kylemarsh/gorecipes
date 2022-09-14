package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func readConfiguration(c *configuration, configFilename string) error {
	file, err := os.Open(configFilename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&c)
}

func jwtGenerate() (string, error) {

	// 1 month expiration. TODO Decide on final scheme?
	claims := &jwt.StandardClaims{ExpiresAt: time.Now().Add(time.Hour * 24 * 30).Unix()}
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

func jwtValidate(tokenString string) error {
	if tokenString == "" {
		return errors.New("missing auth token")
	}

	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.JwtSecret), nil
	})

	return err
}
