package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
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

func TestFlagRecipeCookedSuccess(t *testing.T) {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()
	bootstrap(true)

	// Create a recipe and set it to new
	recipe, _ := createRecipe("Test Recipe", "Body", 10, 20)
	setRecipeNewFlag(recipe.ID, true)

	// Create request to mark it cooked
	req := httptest.NewRequest("PUT", fmt.Sprintf("/recipe/%d/mark_cooked", recipe.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprint(recipe.ID)})
	rr := httptest.NewRecorder()

	// Call handler
	err := flagRecipeCooked(rr, req)

	// Check no appError returned
	if err != nil {
		t.Errorf("flagRecipeCooked() returned appError: %v", err)
	}

	// Check status code
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("flagRecipeCooked() returned wrong status: got %v want %v", status, http.StatusNoContent)
	}

	// Verify database was updated
	updated, _ := recipeByID(recipe.ID, false)
	if updated.New {
		t.Errorf("After flagRecipeCooked(), expected New=false, got New=true")
	}
}

func TestFlagRecipeCookedInvalidID(t *testing.T) {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()
	bootstrap(true)

	// Create request with non-integer ID
	req := httptest.NewRequest("PUT", "/recipe/abc/mark_cooked", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	rr := httptest.NewRecorder()

	// Call handler
	err := flagRecipeCooked(rr, req)

	// Check appError returned
	if err == nil {
		t.Errorf("flagRecipeCooked() with invalid ID should return appError")
	}

	if err != nil && err.Code != http.StatusBadRequest {
		t.Errorf("flagRecipeCooked() with invalid ID returned wrong code: got %v want %v", err.Code, http.StatusBadRequest)
	}
}

func TestFlagRecipeCookedNonExistent(t *testing.T) {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()
	bootstrap(true)

	// Create request with non-existent recipe ID
	req := httptest.NewRequest("PUT", "/recipe/9999/mark_cooked", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "9999"})
	rr := httptest.NewRecorder()

	// Call handler
	err := flagRecipeCooked(rr, req)

	// Check appError returned
	if err == nil {
		t.Errorf("flagRecipeCooked() with non-existent recipe should return appError")
	}

	if err != nil && err.Code != http.StatusNotFound {
		t.Errorf("flagRecipeCooked() with non-existent recipe returned wrong code: got %v want %v", err.Code, http.StatusNotFound)
	}
}

func TestUnFlagRecipeCookedSuccess(t *testing.T) {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()
	bootstrap(true)

	// Create a recipe (defaults to new=false)
	recipe, _ := createRecipe("Test Recipe", "Body", 10, 20)

	// Create request to mark it new
	req := httptest.NewRequest("PUT", fmt.Sprintf("/recipe/%d/mark_new", recipe.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprint(recipe.ID)})
	rr := httptest.NewRecorder()

	// Call handler
	err := unFlagRecipeCooked(rr, req)

	// Check no appError returned
	if err != nil {
		t.Errorf("unFlagRecipeCooked() returned appError: %v", err)
	}

	// Check status code
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("unFlagRecipeCooked() returned wrong status: got %v want %v", status, http.StatusNoContent)
	}

	// Verify database was updated
	updated, _ := recipeByID(recipe.ID, false)
	if !updated.New {
		t.Errorf("After unFlagRecipeCooked(), expected New=true, got New=false")
	}
}

func TestUnFlagRecipeCookedInvalidID(t *testing.T) {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()
	bootstrap(true)

	// Create request with non-integer ID
	req := httptest.NewRequest("PUT", "/recipe/xyz/mark_new", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "xyz"})
	rr := httptest.NewRecorder()

	// Call handler
	err := unFlagRecipeCooked(rr, req)

	// Check appError returned
	if err == nil {
		t.Errorf("unFlagRecipeCooked() with invalid ID should return appError")
	}

	if err != nil && err.Code != http.StatusBadRequest {
		t.Errorf("unFlagRecipeCooked() with invalid ID returned wrong code: got %v want %v", err.Code, http.StatusBadRequest)
	}
}

func TestUnFlagRecipeCookedNonExistent(t *testing.T) {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()
	bootstrap(true)

	// Create request with non-existent recipe ID
	req := httptest.NewRequest("PUT", "/recipe/8888/mark_new", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "8888"})
	rr := httptest.NewRecorder()

	// Call handler
	err := unFlagRecipeCooked(rr, req)

	// Check appError returned
	if err == nil {
		t.Errorf("unFlagRecipeCooked() with non-existent recipe should return appError")
	}

	if err != nil && err.Code != http.StatusNotFound {
		t.Errorf("unFlagRecipeCooked() with non-existent recipe returned wrong code: got %v want %v", err.Code, http.StatusNotFound)
	}
}

func TestRecipeNewFlagIntegration(t *testing.T) {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()
	bootstrap(true)

	// Create a new recipe
	recipe, err := createRecipe("Integration Test Recipe", "Test body", 15, 25)
	if err != nil {
		t.Fatalf("Failed to create recipe: %v", err)
	}

	// Initial state should be new=false
	fetched, _ := recipeByID(recipe.ID, false)
	if fetched.New {
		t.Errorf("Newly created recipe should have New=false, got New=true")
	}

	// Mark as new
	req := httptest.NewRequest("PUT", "/mark_new", nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprint(recipe.ID)})
	rr := httptest.NewRecorder()
	if err := unFlagRecipeCooked(rr, req); err != nil {
		t.Fatalf("unFlagRecipeCooked failed: %v", err)
	}

	// Verify it's new
	fetched, _ = recipeByID(recipe.ID, false)
	if !fetched.New {
		t.Errorf("After marking new, expected New=true, got New=false")
	}

	// Mark as cooked
	req = httptest.NewRequest("PUT", "/mark_cooked", nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprint(recipe.ID)})
	rr = httptest.NewRecorder()
	if err := flagRecipeCooked(rr, req); err != nil {
		t.Fatalf("flagRecipeCooked failed: %v", err)
	}

	// Verify it's not new
	fetched, _ = recipeByID(recipe.ID, false)
	if fetched.New {
		t.Errorf("After marking cooked, expected New=false, got New=true")
	}

	// Toggle back to new
	req = httptest.NewRequest("PUT", "/mark_new", nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprint(recipe.ID)})
	rr = httptest.NewRecorder()
	if err := unFlagRecipeCooked(rr, req); err != nil {
		t.Fatalf("Second unFlagRecipeCooked failed: %v", err)
	}

	// Verify it's new again
	fetched, _ = recipeByID(recipe.ID, false)
	if !fetched.New {
		t.Errorf("After second marking new, expected New=true, got New=false")
	}
}

func TestUpdateExistingRecipeWithNewFlagTrue(t *testing.T) {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()
	bootstrap(true)

	// Create a recipe (defaults to new=false)
	recipe, _ := createRecipe("Test Recipe", "Original Body", 10, 20)

	// Verify initial state
	fetched, _ := recipeByID(recipe.ID, false)
	if fetched.New {
		t.Errorf("Initial recipe should have New=false, got New=true")
	}

	// Create PUT request with new=on (checkbox checked)
	req := httptest.NewRequest("PUT", fmt.Sprintf("/recipe/%d", recipe.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprint(recipe.ID)})
	req.Form = map[string][]string{
		"title":      {"Updated Title"},
		"body":       {"Updated Body"},
		"activeTime": {"15"},
		"totalTime":  {"25"},
		"new":        {"on"},
	}
	rr := httptest.NewRecorder()

	// Call handler
	err := updateExistingRecipe(rr, req)

	// Check no appError returned
	if err != nil {
		t.Errorf("updateExistingRecipe() returned appError: %v", err)
	}

	// Check status code
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("updateExistingRecipe() returned wrong status: got %v want %v", status, http.StatusNoContent)
	}

	// Verify database was updated with new=true
	updated, _ := recipeByID(recipe.ID, false)
	if !updated.New {
		t.Errorf("After update with new=on, expected New=true, got New=false")
	}
	if updated.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", updated.Title)
	}
	if updated.Body != "Updated Body" {
		t.Errorf("Expected body 'Updated Body', got '%s'", updated.Body)
	}
	if updated.ActiveTime != 15 {
		t.Errorf("Expected activeTime 15, got %d", updated.ActiveTime)
	}
	if updated.Time != 25 {
		t.Errorf("Expected totalTime 25, got %d", updated.Time)
	}
}

func TestUpdateExistingRecipeWithNewFlagFalse(t *testing.T) {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()
	bootstrap(true)

	// Create a recipe and set it to new
	recipe, _ := createRecipe("Test Recipe", "Original Body", 10, 20)
	setRecipeNewFlag(recipe.ID, true)

	// Verify initial state
	fetched, _ := recipeByID(recipe.ID, false)
	if !fetched.New {
		t.Errorf("Recipe should have New=true after setRecipeNewFlag, got New=false")
	}

	// Create PUT request WITHOUT new field (checkbox unchecked)
	req := httptest.NewRequest("PUT", fmt.Sprintf("/recipe/%d", recipe.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprint(recipe.ID)})
	req.Form = map[string][]string{
		"title":      {"Updated Title"},
		"body":       {"Updated Body"},
		"activeTime": {"15"},
		"totalTime":  {"25"},
		// "new" field absent - checkbox unchecked
	}
	rr := httptest.NewRecorder()

	// Call handler
	err := updateExistingRecipe(rr, req)

	// Check no appError returned
	if err != nil {
		t.Errorf("updateExistingRecipe() returned appError: %v", err)
	}

	// Check status code
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("updateExistingRecipe() returned wrong status: got %v want %v", status, http.StatusNoContent)
	}

	// Verify database was updated with new=false
	updated, _ := recipeByID(recipe.ID, false)
	if updated.New {
		t.Errorf("After update without new field, expected New=false, got New=true")
	}
	if updated.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", updated.Title)
	}
}

func TestUpdateExistingRecipeNonExistent(t *testing.T) {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()
	bootstrap(true)

	// Create PUT request for non-existent recipe
	req := httptest.NewRequest("PUT", "/recipe/99999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "99999"})
	req.Form = map[string][]string{
		"title":      {"Updated Title"},
		"body":       {"Updated Body"},
		"activeTime": {"15"},
		"totalTime":  {"25"},
		"new":        {"on"},
	}
	rr := httptest.NewRecorder()

	// Call handler
	err := updateExistingRecipe(rr, req)

	// Check appError returned
	if err == nil {
		t.Errorf("updateExistingRecipe() with non-existent recipe should return appError")
	}

	if err != nil && err.Code != http.StatusNotFound {
		t.Errorf("updateExistingRecipe() with non-existent recipe returned wrong code: got %v want %v", err.Code, http.StatusNotFound)
	}
}
