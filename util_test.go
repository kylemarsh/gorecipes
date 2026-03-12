package main

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func setupJwtConfig() {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "test-secret-key-for-jwt",
	}
}

func TestJwtGenerate(t *testing.T) {
	setupJwtConfig()

	token, err := jwtGenerate()
	if err != nil {
		t.Errorf("jwtGenerate() returned error: %v", err)
	}

	if token == "" {
		t.Error("jwtGenerate() returned empty token")
	}

	// Verify the token is parseable
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.JwtSecret), nil
	})

	if err != nil {
		t.Errorf("Generated token could not be parsed: %v", err)
	}

	if !parsed.Valid {
		t.Error("Generated token is not valid")
	}
}

func TestJwtGenerateHasExpirationClaim(t *testing.T) {
	setupJwtConfig()

	tokenString, err := jwtGenerate()
	if err != nil {
		t.Fatalf("jwtGenerate() returned error: %v", err)
	}

	// Parse and check claims
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.JwtSecret), nil
	})

	if err != nil {
		t.Fatalf("Could not parse token: %v", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		t.Fatal("Could not extract RegisteredClaims from token")
	}

	if claims.ExpiresAt == nil {
		t.Error("Token does not have ExpiresAt claim")
	}

	// Verify expiration is approximately 30 days from now
	expectedExpiration := time.Now().Add(time.Hour * 24 * 30)
	actualExpiration := claims.ExpiresAt.Time

	// Allow 5 second variance for test execution time
	timeDiff := actualExpiration.Sub(expectedExpiration)
	if timeDiff < -5*time.Second || timeDiff > 5*time.Second {
		t.Errorf("Token expiration not as expected. Got %v, expected approximately %v (diff: %v)",
			actualExpiration, expectedExpiration, timeDiff)
	}
}

func TestJwtValidateWithValidToken(t *testing.T) {
	setupJwtConfig()

	tokenString, err := jwtGenerate()
	if err != nil {
		t.Fatalf("jwtGenerate() returned error: %v", err)
	}

	err = jwtValidate(tokenString)
	if err != nil {
		t.Errorf("jwtValidate() failed for valid token: %v", err)
	}
}

func TestJwtValidateWithEmptyToken(t *testing.T) {
	setupJwtConfig()

	err := jwtValidate("")
	if err == nil {
		t.Error("jwtValidate() should fail for empty token")
	}

	expectedError := "missing auth token"
	if err.Error() != expectedError {
		t.Errorf("jwtValidate() error message = %v, want %v", err.Error(), expectedError)
	}
}

func TestJwtValidateWithInvalidToken(t *testing.T) {
	setupJwtConfig()

	err := jwtValidate("not.a.valid.token")
	if err == nil {
		t.Error("jwtValidate() should fail for invalid token")
	}
}

func TestJwtValidateWithWrongSecret(t *testing.T) {
	setupJwtConfig()

	// Generate token with one secret
	tokenString, err := jwtGenerate()
	if err != nil {
		t.Fatalf("jwtGenerate() returned error: %v", err)
	}

	// Change the secret and try to validate
	conf.JwtSecret = "different-secret"
	err = jwtValidate(tokenString)
	if err == nil {
		t.Error("jwtValidate() should fail when secret doesn't match")
	}
}

func TestJwtValidateWithExpiredToken(t *testing.T) {
	setupJwtConfig()

	// Create an expired token
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, err := token.SignedString([]byte(conf.JwtSecret))
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	err = jwtValidate(expiredToken)
	if err == nil {
		t.Error("jwtValidate() should fail for expired token")
	}
}

func TestJwtValidateWithDifferentSigningMethod(t *testing.T) {
	setupJwtConfig()

	// Create token with HS384 instead of HS256
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	tokenString, err := token.SignedString([]byte(conf.JwtSecret))
	if err != nil {
		t.Fatalf("Failed to create token with HS384: %v", err)
	}

	// This should still validate since we only check the secret
	err = jwtValidate(tokenString)
	if err != nil {
		t.Errorf("jwtValidate() failed for token with different signing method: %v", err)
	}
}

func TestHashPassword(t *testing.T) {
	password := "testPassword123"
	hashed, err := hashPassword(password)
	if err != nil {
		t.Errorf("hashPassword() returned error: %v", err)
	}

	if hashed == "" {
		t.Error("hashPassword() returned empty string")
	}

	if hashed == password {
		t.Error("hashPassword() returned plaintext password")
	}

	// Hash the same password again - should get different hash (bcrypt uses salt)
	hashed2, err := hashPassword(password)
	if err != nil {
		t.Errorf("Second hashPassword() returned error: %v", err)
	}

	if hashed == hashed2 {
		t.Error("hashPassword() should return different hashes for same password (bcrypt salting)")
	}
}

func TestValidateIcon(t *testing.T) {
	tests := []struct {
		name    string
		icon    string
		wantErr bool
	}{
		{"empty string is valid", "", false},
		{"single ASCII char", "G", false},
		{"single emoji", "🐓", false},
		{"emoji with modifier", "🌶️", false},
		{"country flag", "🇲🇽", false},
		{"circled letter", "Ⓥ", false},
		{"multiple emojis", "🐓🐄", true},
		{"two ASCII chars", "GF", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIcon(tt.icon)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIcon(%q) error = %v, wantErr %v", tt.icon, err, tt.wantErr)
			}
		})
	}
}
