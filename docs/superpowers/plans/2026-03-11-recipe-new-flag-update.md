# Recipe "New" Flag Update Endpoint Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add support for updating the recipe "new" flag through the existing `PUT /priv/recipe/{id}` endpoint

**Architecture:** Extend the existing recipe update handler to parse an optional "new" form value (HTML checkbox) and pass it to the model layer. The checkbox sends "on" when checked, nothing when unchecked. Handler interprets presence as true, absence as false.

**Tech Stack:** Go 1.x, gorilla/mux, jmoiron/sqlx, SQLite/MySQL

---

## File Structure

**Files to Modify:**
- `model.go` - Add `isNew` parameter to `updateRecipe()` function
- `privileged.go` - Add "new" form value parsing to `updateExistingRecipe()` handler
- `privileged_test.go` - Add tests for new flag update functionality

**No New Files Required**

---

## Chunk 1: Core Implementation

### Task 1: Update model.go updateRecipe function

**Files:**
- Modify: `model.go:220-230`
- Test: `model_test.go`

- [ ] **Step 1: Write failing test for updateRecipe with new flag**

Add to `model_test.go`:

```go
func TestUpdateRecipeWithNewFlag(t *testing.T) {
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

	// Create a recipe
	recipe, err := createRecipe("Original Title", "Original Body", 10, 20)
	if err != nil {
		t.Fatalf("Failed to create recipe: %v", err)
	}

	// Update with new=true
	err = updateRecipe(recipe.ID, "Updated Title", "Updated Body", 15, 25, true)
	if err != nil {
		t.Fatalf("updateRecipe failed: %v", err)
	}

	// Verify all fields updated including new flag
	updated, err := recipeByID(recipe.ID, false)
	if err != nil {
		t.Fatalf("Failed to fetch updated recipe: %v", err)
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
	if !updated.New {
		t.Errorf("Expected new=true, got new=false")
	}

	// Update with new=false
	err = updateRecipe(recipe.ID, "Final Title", "Final Body", 5, 10, false)
	if err != nil {
		t.Fatalf("Second updateRecipe failed: %v", err)
	}

	// Verify new flag set to false
	updated, err = recipeByID(recipe.ID, false)
	if err != nil {
		t.Fatalf("Failed to fetch recipe after second update: %v", err)
	}

	if updated.New {
		t.Errorf("Expected new=false after second update, got new=true")
	}
	if updated.Title != "Final Title" {
		t.Errorf("Expected title 'Final Title', got '%s'", updated.Title)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestUpdateRecipeWithNewFlag
```

Expected output:
```
# command-line-arguments
./model_test.go:XXX: not enough arguments in call to updateRecipe
        have (int, string, string, int, int, bool)
        want (int, string, string, int, int)
FAIL    command-line-arguments [build failed]
```

- [ ] **Step 3: Update updateRecipe function signature and implementation**

In `model.go`, modify the `updateRecipe` function (around line 220):

```go
func updateRecipe(recipeId int, title string, body string, activeTime int, totalTime int, isNew bool) error {
	q := `UPDATE recipe SET
		title = ?,
		recipe_body = ?,
		active_time = ?,
		total_time = ?,
		new = ?
		WHERE recipe_id = ?`
	connect()
	_, err := db.Exec(q, title, body, activeTime, totalTime, isNew, recipeId)
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestUpdateRecipeWithNewFlag
```

Expected output:
```
=== RUN   TestUpdateRecipeWithNewFlag
--- PASS: TestUpdateRecipeWithNewFlag (X.XXs)
PASS
ok      command-line-arguments  X.XXXs
```

- [ ] **Step 5: Commit model changes**

```bash
git add model.go model_test.go
git commit -m "feat: add isNew parameter to updateRecipe function

Update the updateRecipe function to accept a boolean isNew parameter
and include it in the SQL UPDATE statement. This allows updating the
recipe new flag along with other recipe fields.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Update privileged.go handler to parse new flag

**Files:**
- Modify: `privileged.go:106-131`
- Test: `privileged_test.go`

- [ ] **Step 1: Write failing test for handler with new=true**

Add to `privileged_test.go`:

```go
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
	recipe, err := createRecipe("Test Recipe", "Original Body", 10, 20)
	if err != nil {
		t.Fatalf("Failed to create recipe: %v", err)
	}

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
	err = updateExistingRecipe(rr, req)

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
```

- [ ] **Step 2: Write failing test for handler with new=false (unchecked)**

Add to `privileged_test.go`:

```go
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
	recipe, err := createRecipe("Test Recipe", "Original Body", 10, 20)
	if err != nil {
		t.Fatalf("Failed to create recipe: %v", err)
	}
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
	err = updateExistingRecipe(rr, req)

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
```

- [ ] **Step 2.5: Write failing test for non-existent recipe**

Add to `privileged_test.go`:

```go
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
```

- [ ] **Step 3: Run tests to verify they fail**

Run:
```bash
go test -v -run "TestUpdateExistingRecipeWithNewFlag"
```

Expected output:
```
# command-line-arguments
./privileged.go:125: not enough arguments in call to updateRecipe
        have (int, string, string, int, int)
        want (int, string, string, int, int, bool)
FAIL    command-line-arguments [build failed]
```
(Note: Line number may vary slightly)

- [ ] **Step 4: Update updateExistingRecipe handler**

In `privileged.go`, modify the `updateExistingRecipe` function (around line 106):

```go
func updateExistingRecipe(w http.ResponseWriter, r *http.Request) *appError {
	recipeId, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "recipe ID must be an integer", err}
	}

	// Validate recipe exists before attempting update
	if _, err := recipeByID(recipeId, false); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &appError{http.StatusNotFound, "recipe does not exist", err}
		}
		return &appError{http.StatusInternalServerError, "problem loading recipe", err}
	}

	title := r.FormValue("title")
	if title == "" {
		return &appError{http.StatusBadRequest, "title is required", nil}
	}
	activeTime, err := strconv.Atoi(r.FormValue("activeTime"))
	if err != nil {
		return &appError{http.StatusBadRequest, "activeTime must be an integer", err}
	}
	totalTime, err := strconv.Atoi(r.FormValue("totalTime"))
	if err != nil {
		return &appError{http.StatusBadRequest, "totalTime must be an integer", err}
	}
	body := r.FormValue("body")
	isNew := r.FormValue("new") != ""

	err = updateRecipe(recipeId, title, body, activeTime, totalTime, isNew)
	if err != nil {
		return &appError{http.StatusInternalServerError, "could not update recipe", err}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run:
```bash
go test -v -run "TestUpdateExistingRecipe"
```

Expected output:
```
=== RUN   TestUpdateExistingRecipeWithNewFlagTrue
--- PASS: TestUpdateExistingRecipeWithNewFlagTrue (X.XXs)
=== RUN   TestUpdateExistingRecipeWithNewFlagFalse
--- PASS: TestUpdateExistingRecipeWithNewFlagFalse (X.XXs)
=== RUN   TestUpdateExistingRecipeNonExistent
--- PASS: TestUpdateExistingRecipeNonExistent (X.XXs)
PASS
ok      command-line-arguments  X.XXXs
```

- [ ] **Step 6: Commit handler changes**

```bash
git add privileged.go privileged_test.go
git commit -m "feat: add new flag support to recipe update endpoint

Parse the 'new' form value in updateExistingRecipe handler. Checkbox
checked sends 'on' which evaluates to true, unchecked sends nothing
which evaluates to false. Pass the boolean to updateRecipe function.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Add integration test for combined update scenarios

**Files:**
- Test: `privileged_test.go`

- [ ] **Step 1: Write integration test for toggling new flag via updates**

Add to `privileged_test.go`:

```go
func TestUpdateRecipeNewFlagIntegration(t *testing.T) {
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

	// Create a recipe
	recipe, err := createRecipe("Integration Test Recipe", "Original Body", 10, 20)
	if err != nil {
		t.Fatalf("Failed to create recipe: %v", err)
	}

	// Initial state: new=false
	fetched, _ := recipeByID(recipe.ID, false)
	if fetched.New {
		t.Errorf("Newly created recipe should have New=false, got New=true")
	}

	// Update 1: Set new=true via update endpoint
	req := httptest.NewRequest("PUT", fmt.Sprintf("/recipe/%d", recipe.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprint(recipe.ID)})
	req.Form = map[string][]string{
		"title":      {"First Update"},
		"body":       {"Body 1"},
		"activeTime": {"12"},
		"totalTime":  {"22"},
		"new":        {"on"},
	}
	rr := httptest.NewRecorder()
	if err := updateExistingRecipe(rr, req); err != nil {
		t.Fatalf("First update failed: %v", err)
	}

	fetched, _ = recipeByID(recipe.ID, false)
	if !fetched.New {
		t.Errorf("After first update with new=on, expected New=true, got New=false")
	}

	// Update 2: Keep new=true while updating other fields
	req = httptest.NewRequest("PUT", fmt.Sprintf("/recipe/%d", recipe.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprint(recipe.ID)})
	req.Form = map[string][]string{
		"title":      {"Second Update"},
		"body":       {"Body 2"},
		"activeTime": {"14"},
		"totalTime":  {"24"},
		"new":        {"on"},
	}
	rr = httptest.NewRecorder()
	if err := updateExistingRecipe(rr, req); err != nil {
		t.Fatalf("Second update failed: %v", err)
	}

	fetched, _ = recipeByID(recipe.ID, false)
	if !fetched.New {
		t.Errorf("After second update with new=on, expected New=true, got New=false")
	}
	if fetched.Title != "Second Update" {
		t.Errorf("Expected title 'Second Update', got '%s'", fetched.Title)
	}

	// Update 3: Toggle to new=false
	req = httptest.NewRequest("PUT", fmt.Sprintf("/recipe/%d", recipe.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprint(recipe.ID)})
	req.Form = map[string][]string{
		"title":      {"Third Update"},
		"body":       {"Body 3"},
		"activeTime": {"16"},
		"totalTime":  {"26"},
		// "new" field absent
	}
	rr = httptest.NewRecorder()
	if err := updateExistingRecipe(rr, req); err != nil {
		t.Fatalf("Third update failed: %v", err)
	}

	fetched, _ = recipeByID(recipe.ID, false)
	if fetched.New {
		t.Errorf("After third update without new field, expected New=false, got New=true")
	}
	if fetched.Title != "Third Update" {
		t.Errorf("Expected title 'Third Update', got '%s'", fetched.Title)
	}

	// Update 4: Toggle back to new=true
	req = httptest.NewRequest("PUT", fmt.Sprintf("/recipe/%d", recipe.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprint(recipe.ID)})
	req.Form = map[string][]string{
		"title":      {"Fourth Update"},
		"body":       {"Body 4"},
		"activeTime": {"18"},
		"totalTime":  {"28"},
		"new":        {"1"}, // Test non-empty string value
	}
	rr = httptest.NewRecorder()
	if err := updateExistingRecipe(rr, req); err != nil {
		t.Fatalf("Fourth update failed: %v", err)
	}

	fetched, _ = recipeByID(recipe.ID, false)
	if !fetched.New {
		t.Errorf("After fourth update with new=1, expected New=true, got New=false")
	}
	if fetched.Title != "Fourth Update" {
		t.Errorf("Expected title 'Fourth Update', got '%s'", fetched.Title)
	}
}
```

- [ ] **Step 2: Run integration test to verify it passes**

Run:
```bash
go test -v -run TestUpdateRecipeNewFlagIntegration
```

Expected output:
```
=== RUN   TestUpdateRecipeNewFlagIntegration
--- PASS: TestUpdateRecipeNewFlagIntegration (X.XXs)
PASS
ok      command-line-arguments  X.XXXs
```

- [ ] **Step 3: Run full test suite**

Run:
```bash
go test -v
```

Expected output: All tests pass, including:
- `TestUpdateRecipeWithNewFlag`
- `TestUpdateExistingRecipeWithNewFlagTrue`
- `TestUpdateExistingRecipeWithNewFlagFalse`
- `TestUpdateRecipeNewFlagIntegration`
- All existing tests (auth, flagRecipeCooked, etc.)

- [ ] **Step 4: Commit integration test**

```bash
git add privileged_test.go
git commit -m "test: add integration test for recipe new flag updates

Add comprehensive integration test covering multiple update scenarios:
- Setting new=true via update
- Keeping new=true while updating other fields
- Toggling to new=false
- Toggling back to new=true with alternate value

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Chunk 2: Documentation and Verification

### Task 4: Update documentation

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Read current README authenticated requests section**

Run:
```bash
grep -n -A 5 "Authenticated Requests" README.md
```

Expected to see lines 53-58 showing existing authenticated endpoints (Get recipe, Delete recipe, Mark as cooked, Mark as new).

- [ ] **Step 2: Add update recipe documentation**

Add a new bullet point to the "Authenticated Requests" section in `README.md` after line 56 (after "Get full recipe (all recipes)"):

```markdown
- Update recipe: `curl -X PUT -H "x-access-token: $TOKEN" -F"title=Recipe Title" -F"body=Recipe body text" -F"activeTime=15" -F"totalTime=30" -F"new=on" http://localhost:8080/priv/recipe/$RECIPE_ID`
```

(Note: The `-F"new=on"` parameter is optional - omit it to mark recipe as cooked. But don't add this note to the README; keep the format minimal like other entries.)

- [ ] **Step 3: Verify documentation format**

Check that the new line follows the same format as existing entries:
- Starts with `- `
- Has descriptive text followed by backtick-wrapped curl command
- Uses `$TOKEN` and `$RECIPE_ID` variables like other examples
- Is properly indented

- [ ] **Step 4: Commit documentation**

```bash
git add README.md
git commit -m "docs: document recipe update endpoint

Add documentation for PUT /priv/recipe/{id} endpoint including the
optional 'new' parameter. When new=on is included, recipe is marked
as new; when omitted, recipe is marked as cooked.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Final verification

**Files:**
- All modified files

- [ ] **Step 1: Run complete test suite**

Run:
```bash
go test -v ./...
```

Expected output: All tests pass across all packages

- [ ] **Step 2: Build the application**

Run:
```bash
go build
```

Expected output: No compilation errors, `gorecipes` binary created

- [ ] **Step 3: Manual smoke test with dev server**

Run:
```bash
./gorecipes mem.config
```

In another terminal, test the endpoint:
```bash
# Get auth token first
TOKEN=$(curl -s -X POST http://localhost:8000/login \
  -d "username=admin&password=admin" | jq -r .token)

# Create a test recipe
RECIPE_ID=$(curl -s -X POST http://localhost:8000/priv/recipe \
  -H "x-access-token: $TOKEN" \
  -d "title=Test Recipe&body=Test Body&activeTime=10&totalTime=20" \
  | jq -r .recipe_id)

# Update with new=on
curl -X PUT "http://localhost:8000/priv/recipe/$RECIPE_ID" \
  -H "x-access-token: $TOKEN" \
  -d "title=Updated Recipe&body=Updated Body&activeTime=15&totalTime=25&new=on"

# Verify new=true
curl -s "http://localhost:8000/priv/recipe/$RECIPE_ID" \
  -H "x-access-token: $TOKEN" | jq '.new'
# Expected: true

# Update without new field
curl -X PUT "http://localhost:8000/priv/recipe/$RECIPE_ID" \
  -H "x-access-token: $TOKEN" \
  -d "title=Updated Recipe 2&body=Updated Body 2&activeTime=20&totalTime=30"

# Verify new=false
curl -s "http://localhost:8000/priv/recipe/$RECIPE_ID" \
  -H "x-access-token: $TOKEN" | jq '.new'
# Expected: false
```

- [ ] **Step 4: Stop dev server**

Press Ctrl+C to stop the dev server

---

## Success Criteria

- [x] `updateRecipe()` accepts `isNew` parameter and updates database
- [x] `updateExistingRecipe()` parses "new" form value correctly
- [x] Checkbox behavior: "on" = true, absent = false
- [x] All tests pass (unit + integration)
- [x] Application builds without errors
- [x] Manual testing confirms correct behavior
- [x] Documentation updated
- [x] All changes committed with descriptive messages

---

## Notes

**Test-Driven Development:** Each task follows strict TDD:
1. Write failing test
2. Run to verify failure
3. Implement minimal code
4. Run to verify success
5. Commit

**Backward Compatibility:** Existing functionality unchanged. The "new" parameter is purely additive. Dedicated `mark_cooked`/`mark_new` endpoints remain available.

**Edge Cases Handled:**
- Missing "new" parameter → false
- "new=on" (standard checkbox) → true
- "new=1" (non-empty string value) → true
- "new=" (empty value) → false

**Skills Referenced:**
- @superpowers:test-driven-development for TDD workflow
- @superpowers:verification-before-completion for final checks
