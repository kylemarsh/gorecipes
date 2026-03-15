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

	token, err := jwtGenerate(1, true)
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

	tokenString, err := jwtGenerate(1, true)
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

func TestValidateType(t *testing.T) {
	tests := []struct {
		name      string
		labelType string
		wantErr   bool
	}{
		{"empty string is valid", "", false},
		{"single char", "c", false},
		{"20 chars (boundary)", "12345678901234567890", false},
		{"21 chars (too long)", "123456789012345678901", true},
		{"lowercase", "course", false},
		{"mixed case", "Course", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateType(tt.labelType)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateType(%q) error = %v, wantErr %v", tt.labelType, err, tt.wantErr)
			}
		})
	}
}

func TestCustomClaimsStructure(t *testing.T) {
	claims := &CustomClaims{
		UserID:  1,
		IsAdmin: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	if claims.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", claims.UserID)
	}
	if claims.IsAdmin != true {
		t.Errorf("Expected IsAdmin true, got %v", claims.IsAdmin)
	}
}

func TestJwtGenerateWithUserInfo(t *testing.T) {
	// Setup test config
	conf.JwtSecret = "test-secret-key-for-testing"

	// Test admin user
	tokenStr, err := jwtGenerate(1, true)
	if err != nil {
		t.Fatalf("jwtGenerate failed: %v", err)
	}
	if tokenStr == "" {
		t.Error("Expected non-empty token string")
	}

	// Parse and verify claims
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.JwtSecret), nil
	})
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		t.Fatal("Failed to cast claims to CustomClaims")
	}

	if claims.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", claims.UserID)
	}
	if claims.IsAdmin != true {
		t.Errorf("Expected IsAdmin true, got %v", claims.IsAdmin)
	}

	// Test non-admin user
	tokenStr2, err := jwtGenerate(2, false)
	if err != nil {
		t.Fatalf("jwtGenerate failed for non-admin: %v", err)
	}

	token2, _ := jwt.ParseWithClaims(tokenStr2, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.JwtSecret), nil
	})
	claims2 := token2.Claims.(*CustomClaims)

	if claims2.UserID != 2 {
		t.Errorf("Expected UserID 2, got %d", claims2.UserID)
	}
	if claims2.IsAdmin != false {
		t.Errorf("Expected IsAdmin false, got %v", claims2.IsAdmin)
	}
}

func TestJwtExtractClaimsValid(t *testing.T) {
	conf.JwtSecret = "test-secret-key-for-testing"

	// Generate a valid token
	tokenStr, _ := jwtGenerate(1, true)

	// Extract claims
	claims, err := jwtExtractClaims(tokenStr)
	if err != nil {
		t.Fatalf("jwtExtractClaims failed: %v", err)
	}

	if claims.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", claims.UserID)
	}
	if claims.IsAdmin != true {
		t.Errorf("Expected IsAdmin true, got %v", claims.IsAdmin)
	}
}

func TestJwtExtractClaimsEmpty(t *testing.T) {
	_, err := jwtExtractClaims("")
	if err == nil {
		t.Error("Expected error for empty token string")
	}
	if err.Error() != "missing auth token" {
		t.Errorf("Expected 'missing auth token' error, got %v", err)
	}
}

func TestJwtExtractClaimsInvalid(t *testing.T) {
	conf.JwtSecret = "test-secret-key-for-testing"

	_, err := jwtExtractClaims("invalid.token.string")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestJwtExtractClaimsExpired(t *testing.T) {
	conf.JwtSecret = "test-secret-key-for-testing"

	// Generate expired token
	claims := &CustomClaims{
		UserID:  1,
		IsAdmin: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(conf.JwtSecret))

	_, err := jwtExtractClaims(tokenStr)
	if err == nil {
		t.Error("Expected error for expired token")
	}
}
