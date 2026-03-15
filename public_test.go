package main

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestLoginReturnsTokenWithAdminClaims(t *testing.T) {
	// Setup: Bootstrap test database
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "test-secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()
	bootstrap(true)

	// Test login as admin user (foo)
	form := url.Values{}
	form.Add("username", "foo")
	form.Add("password", "bar")

	req := httptest.NewRequest("POST", "/login/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	err := login(w, req)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Parse response
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	tokenStr, ok := response["token"].(string)
	if !ok || tokenStr == "" {
		t.Fatal("Expected token in response")
	}

	// Decode JWT and verify claims
	token, _ := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.JwtSecret), nil
	})

	claims := token.Claims.(*CustomClaims)
	if claims.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", claims.UserID)
	}
	if claims.IsAdmin != true {
		t.Errorf("Expected IsAdmin true for user foo, got %v", claims.IsAdmin)
	}
}

func TestLoginReturnsTokenWithNonAdminClaims(t *testing.T) {
	// Setup: Bootstrap test database
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "test-secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()
	bootstrap(true)

	// Test login as non-admin user (koko)
	form := url.Values{}
	form.Add("username", "koko")
	form.Add("password", "cooking for mama")

	req := httptest.NewRequest("POST", "/login/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	err := login(w, req)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Parse response
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	tokenStr := response["token"].(string)

	// Decode JWT and verify claims
	token, _ := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.JwtSecret), nil
	})

	claims := token.Claims.(*CustomClaims)
	if claims.UserID != 2 {
		t.Errorf("Expected UserID 2, got %d", claims.UserID)
	}
	if claims.IsAdmin != false {
		t.Errorf("Expected IsAdmin false for user koko, got %v", claims.IsAdmin)
	}
}
