package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func setupIntegrationTest() {
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
}

func setupTestRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	// Public routes
	router.Handle("/login/", wrappedHandler(login)).Methods("POST")

	// Read-only authenticated routes
	privRouter := router.PathPrefix("/priv").Subrouter()
	privRouter.Use(authRequired)
	privRouter.Handle("/recipes/", wrappedHandler(getAllRecipes)).Methods("GET")

	// Admin-only mutating routes
	adminRouter := router.PathPrefix("/admin").Subrouter()
	adminRouter.Use(authRequired)
	adminRouter.Use(adminRequired)
	adminRouter.Handle("/recipe/", wrappedHandler(createNewRecipe)).Methods("POST")

	return router
}

// TestPrivRouteWithNonAdminToken verifies that non-admin users can access GET /priv/recipes/
func TestPrivRouteWithNonAdminToken(t *testing.T) {
	setupIntegrationTest()

	// Setup router and make request
	router := setupTestRouter()

	// Generate valid token for non-admin user
	tokenStr, err := jwtGenerate(2, false)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	req := httptest.NewRequest("GET", "/priv/recipes/", nil)
	req.Header.Set("x-access-token", tokenStr)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for non-admin access to /priv/recipes/, got %d", w.Code)
	}
}

// TestAdminRouteWithNonAdminToken verifies that non-admin users get 403 on POST /admin/recipe/
func TestAdminRouteWithNonAdminToken(t *testing.T) {
	setupIntegrationTest()

	router := setupTestRouter()

	// Generate valid token for non-admin user (koko, ID=2)
	tokenStr, err := jwtGenerate(2, false)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	req := httptest.NewRequest("POST", "/admin/recipe/", strings.NewReader(""))
	req.Header.Set("x-access-token", tokenStr)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for non-admin access to /admin/recipe/, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "admin access required") {
		t.Errorf("Expected 'admin access required' in response, got %q", body)
	}
}

// TestAdminRouteWithAdminToken verifies that admin users can access POST /admin/recipe/
func TestAdminRouteWithAdminToken(t *testing.T) {
	setupIntegrationTest()

	router := setupTestRouter()

	// Generate valid token for admin user (foo, ID=1)
	tokenStr, err := jwtGenerate(1, true)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Note: This will fail because we're not providing proper recipe data,
	// but the important thing is that it passes the admin authorization check
	recipeJSON := `{"title": "Test Recipe", "body": "Instructions", "total_time": 30, "active_time": 15}`
	req := httptest.NewRequest("POST", "/admin/recipe/", strings.NewReader(recipeJSON))
	req.Header.Set("x-access-token", tokenStr)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Admin should be able to pass the authorization check
	// (Response might be 400 due to invalid data, but not 403 Forbidden)
	if w.Code == http.StatusForbidden {
		t.Errorf("Expected non-403 status for admin access to /admin/recipe/, got 403 Forbidden")
	}
}

// TestAdminRouteWithoutToken verifies that missing token gets 401
func TestAdminRouteWithoutToken(t *testing.T) {
	setupIntegrationTest()

	router := setupTestRouter()

	req := httptest.NewRequest("POST", "/admin/recipe/", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	// Note: no x-access-token header

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing token on /admin/recipe/, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "missing auth token") {
		t.Errorf("Expected 'missing auth token' in response, got %q", body)
	}
}
