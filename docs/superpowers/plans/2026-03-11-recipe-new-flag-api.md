# Recipe "New" Flag API Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add authenticated API endpoints to mark recipes as cooked or new by updating the `new` field.

**Architecture:** Follows existing TDD pattern with model layer (`setRecipeNewFlag`), handler layer (`flagRecipeCooked`, `unFlagRecipeCooked`), and route registration. Mirrors the existing note flagging implementation.

**Tech Stack:** Go 1.24, gorilla/mux, jmoiron/sqlx, Go testing stdlib

---

## Chunk 1: Model Layer

### Task 1: Add setRecipeNewFlag Model Method

**Files:**
- Test: `model_test.go` (add new test function)
- Modify: `model.go` (add new function after `unDeleteRecipe` around line 259)

- [ ] **Step 1: Write the failing test**

Add to `model_test.go`:

```go
func TestSetRecipeNewFlag(t *testing.T) {
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

	// Create a test recipe
	recipe, err := createRecipe("Test Recipe", "Test body", 10, 20)
	if err != nil {
		t.Fatalf("Failed to create test recipe: %v", err)
	}

	// Recipe should default to new=false
	if recipe.New {
		t.Errorf("New recipe should have New=false by default, got New=true")
	}

	// Set to new (true)
	err = setRecipeNewFlag(recipe.ID, true)
	if err != nil {
		t.Errorf("setRecipeNewFlag(true) returned error: %v", err)
	}

	// Verify it was set
	updated, err := recipeByID(recipe.ID, false)
	if err != nil {
		t.Fatalf("Failed to fetch recipe after update: %v", err)
	}
	if !updated.New {
		t.Errorf("After setRecipeNewFlag(true), expected New=true, got New=false")
	}

	// Set to cooked (false)
	err = setRecipeNewFlag(recipe.ID, false)
	if err != nil {
		t.Errorf("setRecipeNewFlag(false) returned error: %v", err)
	}

	// Verify it was set
	updated, err = recipeByID(recipe.ID, false)
	if err != nil {
		t.Fatalf("Failed to fetch recipe after second update: %v", err)
	}
	if updated.New {
		t.Errorf("After setRecipeNewFlag(false), expected New=false, got New=true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestSetRecipeNewFlag
```

Expected output: `FAIL` with "undefined: setRecipeNewFlag"

- [ ] **Step 3: Write minimal implementation**

Add to `model.go` after `unDeleteRecipe` (around line 259):

```go
func setRecipeNewFlag(recipeID int, isNew bool) error {
	q := "UPDATE recipe SET new = ? WHERE recipe_id = ?"
	connect()
	_, err := db.Exec(q, isNew, recipeID)
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestSetRecipeNewFlag
```

Expected output: `PASS`

- [ ] **Step 5: Commit**

```bash
git add model.go model_test.go
git commit -m "Add setRecipeNewFlag model method

Adds database method to update the 'new' field on recipes. Used to mark
recipes as cooked (new=false) or new/uncooked (new=true).

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Chunk 2: Handler Layer

### Task 2: Add flagRecipeCooked Handler

**Files:**
- Test: `privileged_test.go` (add new test functions)
- Modify: `privileged.go` (add handler after `recipeRestore` around line 348)

- [ ] **Step 1: Write the failing test**

Add to `privileged_test.go`:

```go
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
```

Add import for `fmt` at the top of `privileged_test.go` if not already present.

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test -v -run TestFlagRecipeCooked
```

Expected output: `FAIL` with "undefined: flagRecipeCooked"

- [ ] **Step 3: Write minimal implementation**

Add to `privileged.go` after `recipeRestore` (around line 348):

```go
func flagRecipeCooked(w http.ResponseWriter, r *http.Request) *appError {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "recipe ID must be an integer", err}
	}

	if _, err := recipeByID(recipeID, false); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &appError{http.StatusNotFound, "recipe does not exist", err}
		}
		return &appError{http.StatusInternalServerError, "Problem loading recipe", err}
	}

	if err := setRecipeNewFlag(recipeID, false); err != nil {
		return &appError{http.StatusInternalServerError, "problem setting recipe new flag", err}
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:
```bash
go test -v -run TestFlagRecipeCooked
```

Expected output: All 3 tests `PASS`

- [ ] **Step 5: Commit**

```bash
git add privileged.go privileged_test.go
git commit -m "Add flagRecipeCooked handler

Handler marks recipes as cooked by setting new=false. Validates recipe
ID, checks recipe exists, returns 404 for missing recipes and 400 for
invalid IDs.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

### Task 3: Add unFlagRecipeCooked Handler

**Files:**
- Test: `privileged_test.go` (add new test functions)
- Modify: `privileged.go` (add handler after `flagRecipeCooked`)

- [ ] **Step 1: Write the failing test**

Add to `privileged_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test -v -run TestUnFlagRecipeCooked
```

Expected output: `FAIL` with "undefined: unFlagRecipeCooked"

- [ ] **Step 3: Write minimal implementation**

Add to `privileged.go` after `flagRecipeCooked`:

```go
func unFlagRecipeCooked(w http.ResponseWriter, r *http.Request) *appError {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "recipe ID must be an integer", err}
	}

	if _, err := recipeByID(recipeID, false); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &appError{http.StatusNotFound, "recipe does not exist", err}
		}
		return &appError{http.StatusInternalServerError, "Problem loading recipe", err}
	}

	if err := setRecipeNewFlag(recipeID, true); err != nil {
		return &appError{http.StatusInternalServerError, "problem setting recipe new flag", err}
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:
```bash
go test -v -run TestUnFlagRecipeCooked
```

Expected output: All 3 tests `PASS`

- [ ] **Step 5: Commit**

```bash
git add privileged.go privileged_test.go
git commit -m "Add unFlagRecipeCooked handler

Handler marks recipes as new/uncooked by setting new=true. Validates
recipe ID, checks recipe exists, returns 404 for missing recipes and
400 for invalid IDs.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Chunk 3: Route Registration and Bootstrap Data

### Task 4: Register Routes

**Files:**
- Modify: `main.go` (add routes around line 49 after `privRouter.Handle("/recipe/{id}/restore"`)

- [ ] **Step 1: Add route registrations**

Add to `main.go` in the `privRouter` section after line 48:

```go
	privRouter.Handle("/recipe/{id}/mark_cooked", wrappedHandler(flagRecipeCooked)).Methods("PUT")
	privRouter.Handle("/recipe/{id}/mark_new", wrappedHandler(unFlagRecipeCooked)).Methods("PUT")
```

- [ ] **Step 2: Verify compilation**

Run:
```bash
go build
```

Expected output: No errors, binary created successfully

- [ ] **Step 3: Run all tests to ensure routes work**

Run:
```bash
go test -v ./...
```

Expected output: All tests `PASS`

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "Register recipe new flag routes

Adds PUT /priv/recipe/{id}/mark_cooked and PUT /priv/recipe/{id}/mark_new
routes under authenticated privRouter.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

### Task 5: Update Bootstrap Data

**Files:**
- Modify: `bootstrapping/recipes.csv` (update `new` field for selected recipes)

- [ ] **Step 1: Update recipes in CSV**

Edit `bootstrapping/recipes.csv`:
- Change recipe 2 (Butternut Squash and Sage Wontons): last field from `0` to `1`
- Change recipe 5 (Ground Turkey Laap): last field from `0` to `1`
- Change recipe 8 (Mushrooms Stuffed with Prosciutto): last field from `0` to `1`
- Change recipe 11 (Pot Stickers Veggie): last field from `0` to `1`

These are on lines 62, 123, 188, and 281 respectively (the final `0` on each line).

- [ ] **Step 2: Verify bootstrap still works**

Run:
```bash
go test -v -run TestBootstrap
```

Expected output: `PASS` - bootstrap test validates data loads correctly

- [ ] **Step 3: Test manually with dev server**

Run:
```bash
go build && ./gorecipes --config mem.config --debug --bootstrap
```

In another terminal:
```bash
# Get a token
TOKEN=$(curl -s http://localhost:8080/debug/getToken/ | jq -r .token)

# Fetch recipe 2 - should have New=true
curl -H "x-access-token: $TOKEN" http://localhost:8080/priv/recipe/2/ | jq .New

# Fetch recipe 1 - should have New=false
curl -H "x-access-token: $TOKEN" http://localhost:8080/priv/recipe/1/ | jq .New
```

Expected: Recipe 2 shows `true`, Recipe 1 shows `false`

Stop the server with Ctrl+C.

- [ ] **Step 4: Commit**

```bash
git add bootstrapping/recipes.csv
git commit -m "Mark sample recipes as new in bootstrap data

Sets new=true for recipes 2, 5, 8, and 11 to demonstrate the new flag
feature with variety across recipe types.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Chunk 4: Integration Testing

### Task 6: End-to-End Integration Test

**Files:**
- Test: `privileged_test.go` (add integration test)

- [ ] **Step 1: Write integration test**

Add to `privileged_test.go`:

```go
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
```

- [ ] **Step 2: Run integration test**

Run:
```bash
go test -v -run TestRecipeNewFlagIntegration
```

Expected output: `PASS`

- [ ] **Step 3: Run full test suite**

Run:
```bash
go test -v ./...
```

Expected output: All tests `PASS`

- [ ] **Step 4: Commit**

```bash
git add privileged_test.go
git commit -m "Add integration test for recipe new flag

Tests complete workflow: create recipe, toggle new flag multiple times,
verify database state after each operation.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

### Task 7: Manual Testing and Verification

**Files:**
- None (manual testing only)

- [ ] **Step 1: Start development server**

Run:
```bash
go build && ./gorecipes --config mem.config --debug --bootstrap
```

- [ ] **Step 2: Test mark_cooked endpoint**

In another terminal:
```bash
# Get a token
TOKEN=$(curl -s http://localhost:8080/debug/getToken/ | jq -r .token)

# Mark recipe 2 (currently new=true) as cooked
curl -X PUT -H "x-access-token: $TOKEN" -v http://localhost:8080/priv/recipe/2/mark_cooked

# Verify it's now cooked (new=false)
curl -H "x-access-token: $TOKEN" http://localhost:8080/priv/recipe/2/ | jq .New
```

Expected: First request returns `204 No Content`, second request shows `false`

- [ ] **Step 3: Test mark_new endpoint**

```bash
# Mark recipe 1 (currently new=false) as new
curl -X PUT -H "x-access-token: $TOKEN" -v http://localhost:8080/priv/recipe/1/mark_new

# Verify it's now new (new=true)
curl -H "x-access-token: $TOKEN" http://localhost:8080/priv/recipe/1/ | jq .New
```

Expected: First request returns `204 No Content`, second request shows `true`

- [ ] **Step 4: Test error cases**

```bash
# Test with invalid ID (non-integer)
curl -X PUT -H "x-access-token: $TOKEN" -v http://localhost:8080/priv/recipe/abc/mark_cooked

# Test with non-existent ID
curl -X PUT -H "x-access-token: $TOKEN" -v http://localhost:8080/priv/recipe/9999/mark_cooked

# Test without auth token
curl -X PUT -v http://localhost:8080/priv/recipe/1/mark_cooked
```

Expected:
- Invalid ID returns `400 Bad Request`
- Non-existent ID returns `404 Not Found`
- No auth token returns `401 Unauthorized`

- [ ] **Step 5: Stop server and verify**

Stop the server with Ctrl+C.

Verify no errors in server logs during manual testing.

- [ ] **Step 6: Verify manual testing complete**

Confirm all manual tests passed as expected. No further action or commit needed.

---

## Final Verification

- [ ] **Run complete test suite**

Run:
```bash
go test -v ./...
```

Expected: All tests `PASS`

- [ ] **Build and verify no compilation errors**

Run:
```bash
go build
```

Expected: No errors, binary created

- [ ] **Review all changes**

Run:
```bash
git log --oneline -10
git diff main..HEAD
```

Expected: 6 commits matching the feature implementation, clean diff

---

## Implementation Complete

All tasks completed. The feature is ready for:
1. Code review (use @superpowers:requesting-code-review)
2. Integration with frontend
3. Deployment to staging for testing

### API Endpoints Added

- `PUT /priv/recipe/{id}/mark_cooked` - Marks recipe as cooked (new=false)
- `PUT /priv/recipe/{id}/mark_new` - Marks recipe as new/uncooked (new=true)

Both require authentication via `x-access-token` header.
