package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func setupAuthConfig() {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "test-secret-for-auth",
	}
}

// Mock handler to verify request reaches protected endpoint
func mockProtectedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("protected resource"))
}

func TestAuthRequiredWithValidToken(t *testing.T) {
	setupAuthConfig()

	// Generate a valid token
	tokenString, err := jwtGenerate()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create request with valid token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("x-access-token", tokenString)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create handler with authRequired middleware
	handler := authRequired(http.HandlerFunc(mockProtectedHandler))
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("authRequired() with valid token returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check response body
	expected := "protected resource"
	if rr.Body.String() != expected {
		t.Errorf("authRequired() with valid token returned unexpected body: got %q want %q",
			rr.Body.String(), expected)
	}
}

func TestAuthRequiredWithMissingToken(t *testing.T) {
	setupAuthConfig()

	// Create request without token
	req := httptest.NewRequest("GET", "/protected", nil)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create handler with authRequired middleware
	handler := authRequired(http.HandlerFunc(mockProtectedHandler))
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("authRequired() with missing token returned wrong status code: got %v want %v",
			status, http.StatusUnauthorized)
	}

	// Check exact response body (http.Error adds newline)
	expected := "missing auth token\n"
	if rr.Body.String() != expected {
		t.Errorf("authRequired() with missing token returned unexpected body: got %q want %q",
			rr.Body.String(), expected)
	}
}

func TestAuthRequiredWithEmptyToken(t *testing.T) {
	setupAuthConfig()

	// Create request with empty token (whitespace only)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("x-access-token", "   ")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create handler with authRequired middleware
	handler := authRequired(http.HandlerFunc(mockProtectedHandler))
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("authRequired() with empty token returned wrong status code: got %v want %v",
			status, http.StatusUnauthorized)
	}

	// Check exact response body
	expected := "missing auth token\n"
	if rr.Body.String() != expected {
		t.Errorf("authRequired() with empty token returned unexpected body: got %q want %q",
			rr.Body.String(), expected)
	}
}

func TestAuthRequiredWithInvalidToken(t *testing.T) {
	setupAuthConfig()

	// Create request with invalid token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("x-access-token", "invalid.token.here")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create handler with authRequired middleware
	handler := authRequired(http.HandlerFunc(mockProtectedHandler))
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("authRequired() with invalid token returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Check exact response body
	expected := "invalid auth token\n"
	if rr.Body.String() != expected {
		t.Errorf("authRequired() with invalid token returned unexpected body: got %q want %q",
			rr.Body.String(), expected)
	}
}

func TestAuthRequiredWithExpiredToken(t *testing.T) {
	setupAuthConfig()

	// Create an expired token
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, err := token.SignedString([]byte(conf.JwtSecret))
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	// Create request with expired token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("x-access-token", expiredToken)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create handler with authRequired middleware
	handler := authRequired(http.HandlerFunc(mockProtectedHandler))
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("authRequired() with expired token returned wrong status code: got %v want %v",
			status, http.StatusUnauthorized)
	}

	// Check exact response body
	expected := "auth token expired; please log in again\n"
	if rr.Body.String() != expected {
		t.Errorf("authRequired() with expired token returned unexpected body: got %q want %q",
			rr.Body.String(), expected)
	}
}

func TestAuthRequiredWithWrongSecret(t *testing.T) {
	setupAuthConfig()

	// Generate token with current secret
	tokenString, err := jwtGenerate()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Change the secret
	conf.JwtSecret = "different-secret"

	// Create request with token signed by old secret
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("x-access-token", tokenString)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create handler with authRequired middleware
	handler := authRequired(http.HandlerFunc(mockProtectedHandler))
	handler.ServeHTTP(rr, req)

	// Check status code - should be bad request since signature is invalid
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("authRequired() with wrong secret returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Check exact response body
	expected := "invalid auth token\n"
	if rr.Body.String() != expected {
		t.Errorf("authRequired() with wrong secret returned unexpected body: got %q want %q",
			rr.Body.String(), expected)
	}
}

func TestAuthRequiredWithTokenInWhitespace(t *testing.T) {
	setupAuthConfig()

	// Generate a valid token
	tokenString, err := jwtGenerate()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create request with token wrapped in whitespace
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("x-access-token", "  "+tokenString+"  ")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create handler with authRequired middleware
	handler := authRequired(http.HandlerFunc(mockProtectedHandler))
	handler.ServeHTTP(rr, req)

	// Check status code - should succeed since we trim whitespace
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("authRequired() with whitespace around token returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check exact response body
	expected := "protected resource"
	if rr.Body.String() != expected {
		t.Errorf("authRequired() with whitespace around token returned unexpected body: got %q want %q",
			rr.Body.String(), expected)
	}
}
