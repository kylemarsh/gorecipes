# Label Type Attribute Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add type field to labels for categorization (protein, course, cuisine, dietary)

**Architecture:** Add Type field to Label struct, extend updateLabel model method with type validation, update editLabel handler to accept type parameter, update bootstrap data with type column

**Tech Stack:** Go, gorilla/mux, sqlx, MySQL/SQLite3

---

## Chunk 1: Data Model

### Task 1: Update Label Struct

**Files:**
- Modify: `model.go:41-46`

- [ ] **Step 1: Add Type field to Label struct**

Update the Label struct in model.go (around line 41):

```go
/*Label - a taxonomic tag for recipes */
type Label struct {
	ID    int    `db:"label_id"`
	Label string
	Icon  string
	Type  string
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build
```

Expected: No errors (existing code still works with Type field)

- [ ] **Step 3: Commit struct update**

```bash
git add model.go
git commit -m "Add Type field to Label struct

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Add Type Validation Helper and Error Constant

**Files:**
- Modify: `util.go` (add error constant and validation function)
- Modify: `util_test.go`

- [ ] **Step 1: Add ErrTypeValidation error constant**

Add to `util.go` near the existing error constants (around line 16-17):

```go
var (
	ErrIconValidation = errors.New("icon validation failed")
	ErrLabelConflict  = errors.New("label name conflict")
	ErrTypeValidation = errors.New("type validation failed")
)
```

- [ ] **Step 2: Write failing test for type validation**

Add to `util_test.go`:

```go
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
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test -run TestValidateType
```

Expected: FAIL - "undefined: validateType"

- [ ] **Step 4: Implement validateType**

Add to `util.go` after validateIcon function:

```go
func validateType(labelType string) error {
	if labelType == "" {
		return nil // Empty is valid
	}

	if len(labelType) > 20 {
		return fmt.Errorf("type must be 20 characters or less, got %d: %w", len(labelType), ErrTypeValidation)
	}
	return nil
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test -run TestValidateType -v
```

Expected: PASS (all test cases pass)

- [ ] **Step 6: Commit validation function and error constant**

```bash
git add util.go util_test.go
git commit -m "Add validateType helper and ErrTypeValidation constant

Validates type field: empty string or max 20 characters. Wraps errors
with ErrTypeValidation for errors.Is checking.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Chunk 2: Model Method Updates

### Task 3: Update updateLabel Model Method

**Files:**
- Modify: `model.go:270-304` (updateLabel function)
- Modify: `model_test.go`

- [ ] **Step 1: Write failing test for type parameter**

Add to `model_test.go` (find existing TestUpdateLabel or create new):

```go
func TestUpdateLabelWithType(t *testing.T) {
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

	// Test 1: Update type only
	err := updateLabel(1, "chicken", "🐓", "protein")
	if err != nil {
		t.Errorf("updateLabel() error = %v", err)
	}

	label, _ := labelByID(1)
	if label.Type != "protein" {
		t.Errorf("Expected type 'protein', got %q", label.Type)
	}

	// Test 2: Type normalization (uppercase -> lowercase)
	err = updateLabel(1, "chicken", "🐓", "PROTEIN")
	if err != nil {
		t.Errorf("updateLabel() error = %v", err)
	}

	label, _ = labelByID(1)
	if label.Type != "protein" {
		t.Errorf("Expected lowercase 'protein', got %q", label.Type)
	}

	// Test 3: Empty type clears it
	err = updateLabel(1, "chicken", "🐓", "")
	if err != nil {
		t.Errorf("updateLabel() with empty type error = %v", err)
	}

	label, _ = labelByID(1)
	if label.Type != "" {
		t.Errorf("Expected empty type, got %q", label.Type)
	}

	// Test 4: Type too long should fail
	err = updateLabel(1, "chicken", "🐓", "123456789012345678901")
	if err == nil {
		t.Error("Expected error for type too long, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestUpdateLabelWithType
```

Expected: FAIL - "too many arguments in call to updateLabel"

- [ ] **Step 3: Update updateLabel signature and implementation**

Modify `updateLabel` in `model.go` (around line 270):

```go
func updateLabel(labelID int, newName string, icon string, labelType string) error {
	// Validate icon
	if err := validateIcon(icon); err != nil {
		return err
	}

	// Validate type
	if err := validateType(labelType); err != nil {
		return err
	}

	// Fetch existing label to check if it exists (uses labelByID from model.go:103)
	existing, err := labelByID(labelID)
	if err != nil {
		return err // Returns sql.ErrNoRows if not found
	}

	// Normalize new name to lowercase
	normalizedName := strings.ToLower(newName)

	// Normalize type to lowercase
	normalizedType := strings.ToLower(labelType)

	// Check for name conflicts if name is changing
	if normalizedName != existing.Label {
		var count int
		q := "SELECT COUNT(*) FROM label WHERE LOWER(label) = ? AND label_id != ?"
		connect()
		err := db.Get(&count, q, normalizedName, labelID)
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("label name already exists: %s: %w", newName, ErrLabelConflict)
		}
	}

	// Update all three fields
	q := "UPDATE label SET label = ?, icon = ?, type = ? WHERE label_id = ?"
	connect()
	_, err = db.Exec(q, normalizedName, icon, normalizedType, labelID)
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestUpdateLabelWithType -v
```

Expected: PASS

- [ ] **Step 5: Find and fix existing tests that call updateLabel**

Find all tests that call updateLabel:

```bash
grep -n "updateLabel(" model_test.go privileged_test.go
```

Expected output shows these calls in `model_test.go`:
- Line 192: `updateLabel(1, "newname", "🐄")`
- Line 206: `updateLabel(1, "another", "🐓🐄")`
- Line 212: `updateLabel(1, "beef", "🐓")`
- Line 218: `updateLabel(1, "cleared", "")`
- Line 228: `updateLabel(999, "fake", "")`

Update each call to add empty string as 4th parameter:

```go
// model_test.go line 192
updateLabel(1, "newname", "🐄", "")

// model_test.go line 206
updateLabel(1, "another", "🐓🐄", "")

// model_test.go line 212
updateLabel(1, "beef", "🐓", "")

// model_test.go line 218
updateLabel(1, "cleared", "", "")

// model_test.go line 228
updateLabel(999, "fake", "", "")
```

No changes needed in `privileged_test.go` (doesn't call updateLabel directly).

- [ ] **Step 6: Run all model tests**

```bash
go test -v
```

Expected: All tests pass

- [ ] **Step 7: Commit model method update**

```bash
git add model.go model_test.go
git commit -m "Extend updateLabel to accept and validate type parameter

Validates type length (max 20 chars), normalizes to lowercase,
updates type field atomically with name and icon.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Chunk 3: API Handler Updates

### Task 4: Update editLabel Handler

**Files:**
- Modify: `privileged.go:432-481` (editLabel function)
- Modify: `privileged_test.go`

- [ ] **Step 1: Write failing test for type form parameter**

Add to `privileged_test.go`:

```go
func TestEditLabelWithType(t *testing.T) {
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

	// Test 1: Update type only
	req := makeAuthReq("PUT", "/priv/label/id/1", map[string]string{
		"type": "protein",
	})
	resp := sendReq(req, editLabel)
	if resp.Code != 204 {
		t.Errorf("Expected 204, got %d: %s", resp.Code, resp.Body.String())
	}

	label, _ := labelByID(1)
	if label.Type != "protein" {
		t.Errorf("Expected type 'protein', got %q", label.Type)
	}

	// Test 2: Update all three fields
	req = makeAuthReq("PUT", "/priv/label/id/1", map[string]string{
		"label": "beef",
		"icon":  "🐄",
		"type":  "protein",
	})
	resp = sendReq(req, editLabel)
	if resp.Code != 204 {
		t.Errorf("Expected 204, got %d", resp.Code)
	}

	label, _ = labelByID(1)
	if label.Label != "beef" || label.Icon != "🐄" || label.Type != "protein" {
		t.Errorf("Expected beef/🐄/protein, got %q/%q/%q", label.Label, label.Icon, label.Type)
	}

	// Test 3: Type too long returns 400
	req = makeAuthReq("PUT", "/priv/label/id/1", map[string]string{
		"type": "123456789012345678901",
	})
	resp = sendReq(req, editLabel)
	if resp.Code != 400 {
		t.Errorf("Expected 400 for type too long, got %d", resp.Code)
	}

	// Test 4: Clear type with empty string
	req = makeAuthReq("PUT", "/priv/label/id/1", map[string]string{
		"type": "",
	})
	resp = sendReq(req, editLabel)
	if resp.Code != 204 {
		t.Errorf("Expected 204, got %d", resp.Code)
	}

	label, _ = labelByID(1)
	if label.Type != "" {
		t.Errorf("Expected empty type, got %q", label.Type)
	}

	// Test 5: Missing type parameter preserves existing value
	// First set a type
	updateLabel(1, "beef", "🐄", "protein")

	// Then update only icon (no type parameter)
	req = makeAuthReq("PUT", "/priv/label/id/1", map[string]string{
		"icon": "🥩",
	})
	resp = sendReq(req, editLabel)
	if resp.Code != 204 {
		t.Errorf("Expected 204, got %d", resp.Code)
	}

	label, _ = labelByID(1)
	if label.Type != "protein" {
		t.Errorf("Type should be preserved, got %q", label.Type)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestEditLabelWithType
```

Expected: FAIL - tests fail because handler doesn't handle type yet

- [ ] **Step 3: Update editLabel handler**

Modify `editLabel` in `privileged.go` (around line 432):

```go
func editLabel(w http.ResponseWriter, r *http.Request) *appError {
	labelID, err := strconv.Atoi(mux.Vars(r)["label_id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "label ID must be an integer", err}
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		return &appError{http.StatusBadRequest, "invalid form data", err}
	}

	// Get optional form parameters
	newName := r.FormValue("label")
	icon := r.FormValue("icon")
	labelType := r.FormValue("type")

	// Fetch existing label to get current values
	existing, err := labelByID(labelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &appError{http.StatusNotFound, "label does not exist", err}
		}
		return &appError{http.StatusInternalServerError, "problem loading label", err}
	}

	// Use existing values if parameters not provided
	if !r.Form.Has("label") {
		newName = existing.Label
	}
	// Note: icon can be explicitly set to empty string to clear it
	// So we use r.Form.Has to distinguish "not provided" from "empty string"
	if !r.Form.Has("icon") {
		icon = existing.Icon
	}
	if !r.Form.Has("type") {
		labelType = existing.Type
	}

	// Update the label
	err = updateLabel(labelID, newName, icon, labelType)
	if err != nil {
		// Check if it's a validation error
		if errors.Is(err, ErrIconValidation) {
			return &appError{http.StatusBadRequest, err.Error(), err}
		}
		if errors.Is(err, ErrTypeValidation) {
			return &appError{http.StatusBadRequest, err.Error(), err}
		}
		if errors.Is(err, ErrLabelConflict) {
			return &appError{http.StatusConflict, err.Error(), err}
		}
		return &appError{http.StatusInternalServerError, "problem updating label", err}
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestEditLabelWithType -v
```

Expected: PASS

- [ ] **Step 5: Run all privileged tests**

```bash
go test -v
```

Expected: All tests pass

- [ ] **Step 6: Commit handler update**

```bash
git add privileged.go privileged_test.go
git commit -m "Extend editLabel handler to accept type form parameter

Supports partial updates: missing type parameter preserves existing
value. Empty string explicitly clears type.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Chunk 4: Bootstrap Data Updates

### Task 5: Update bootstrapping/labels.csv

**Files:**
- Modify: `bootstrapping/labels.csv`

- [ ] **Step 1: Update labels.csv with type column**

Replace contents of `bootstrapping/labels.csv`:

```csv
"label_id";"label";"icon";"type"
"1";"chicken";"🐓";"protein"
"2";"beef";"🐄";"protein"
"3";"pork";"🐖";"protein"
"4";"fish";"🐟";"protein"
"5";"lamb";"🐑";"protein"
"6";"vegetarian";"🥦";"dietary"
"7";"vegan";"Ⓥ";"dietary"
"8";"breakfast";"🍳";"course"
"9";"drink";"🥤";"course"
"10";"soupstew";"🍜";"dish"
"11";"salad";"🥬";"dish"
"12";"appetizer";"🥟";"course"
"13";"sandwich";"🥪";"dish"
"14";"mexican";"🇲🇽";"cuisine"
"15";"asian";"🥢";"cuisine"
"16";"middleeast";"";"cuisine"
"17";"dessert";"🍦";"course"
"18";"bread";"🍞";"dish"
"19";"vegetable";"🥕";"ingredient"
"20";"cookie";"🍪";"dish"
"21";"cake";"🎂";"dish"
"22";"candy";"🍬";"dish"
"23";"cheesecake";"";"dish"
"24";"creamcustard";"";"dish"
"25";"fruit";"🍏";"ingredient"
"26";"glutenfree";"Ⓖ";"dietary"
"27";"pasta";"🍝";"dish"
"28";"greek";"🇬🇷";"cuisine"
"29";"spicy";"🌶️";"attribute"
"30";"quick";"⚡";"attribute"
"31";"shrimp";"🦐";"protein"
"32";"sauce";"";"dish"
"33";"rice";"🍚";"dish"
"34";"starchside";"";"dish"
"35";"side";"";"course"
"36";"main";"🍽️";"course"
"37";"turkey";"🦃";"protein"
"38";"batch";"⏰";"attribute"
"39";"lamp";"";""
"40";"sousvide";"🌡️";"preparation"
"41";"air fryer";"";"preparation"
"42";"soup";"🥣";"dish"
"43";"summer";"";""
"44";"sauces";"";"dish"
"45";"tofu";"";"protein"
"46";"eatsy";"";""
```

- [ ] **Step 2: Commit CSV update**

```bash
git add bootstrapping/labels.csv
git commit -m "Add type column to bootstrapping labels.csv

Categorizes labels: protein, course, cuisine, dietary, dish,
attribute, ingredient, preparation.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Update bootstrap_recipes.go

**Files:**
- Modify: `bootstrapping/bootstrap_recipes.go:46-52`

- [ ] **Step 1: Update label table definition**

Modify the "label" entry in the info map (around lines 46-52):

```go
"label": {
	"filename":       dir + "labels.csv",
	"drop":           "DROP TABLE IF EXISTS label",
	"create_mysql":   "CREATE TABLE `label` ( `label_id` int(11) NOT NULL auto_increment, `label` varchar(255) NOT NULL, `icon` varchar(255) NOT NULL DEFAULT '', `type` varchar(20) NOT NULL DEFAULT '', PRIMARY KEY  (`label_id`), KEY `label` (`label`))",
	"create_sqlite3": "CREATE TABLE `label` ( `label_id` INTEGER PRIMARY KEY, `label` varchar(255) NOT NULL, `icon` varchar(255) NOT NULL DEFAULT '', `type` varchar(20) NOT NULL DEFAULT '')",
	"insert":         "INSERT INTO label (label_id, label, icon, type) VALUES (?, ?, ?, ?)",
},
```

- [ ] **Step 2: Verify compilation**

```bash
cd bootstrapping
go build
```

Expected: No errors

- [ ] **Step 3: Test standalone bootstrap (optional)**

```bash
cd bootstrapping
rm -f recipes_sqlite.db
./bootstrapping -dialect sqlite3 -dsn recipes_sqlite.db
sqlite3 recipes_sqlite.db "SELECT label_id, label, type FROM label LIMIT 5;"
```

Expected: Shows labels with types (e.g., "1|chicken|protein")

- [ ] **Step 4: Commit bootstrap_recipes.go update**

```bash
git add bootstrapping/bootstrap_recipes.go
git commit -m "Update bootstrap_recipes.go to handle type column

Updates CREATE TABLE and INSERT statements to include type field.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 7: Update bootstrap.go

**Files:**
- Modify: `bootstrap.go:29-35`

- [ ] **Step 1: Update label table definition**

Modify the "label" entry in the info map (around lines 29-35):

```go
"label": {
	"filename":       dir + "bootstrapping/labels.csv",
	"drop":           "DROP TABLE IF EXISTS label",
	"create_mysql":   "CREATE TABLE `label` ( `label_id` int(11) NOT NULL auto_increment, `label` varchar(255) NOT NULL, `icon` varchar(255) NOT NULL DEFAULT '', `type` varchar(20) NOT NULL DEFAULT '', PRIMARY KEY  (`label_id`), KEY `label` (`label`))",
	"create_sqlite3": "CREATE TABLE `label` ( `label_id` INTEGER PRIMARY KEY, `label` varchar(255) NOT NULL, `icon` varchar(255) NOT NULL DEFAULT '', `type` varchar(20) NOT NULL DEFAULT '')",
	"insert":         "INSERT INTO label (label_id, label, icon, type) VALUES (?, ?, ?, ?)",
},
```

- [ ] **Step 2: Test embedded bootstrap**

```bash
go test -run TestBootstrap -v
```

Expected: PASS - bootstrap should now expect 4 columns from labels.csv

- [ ] **Step 3: Test full application with bootstrap**

```bash
go run . -config mem.config -bootstrap &
SERVER_PID=$!
sleep 2

# Check that labels have types
TOKEN=$(curl -s -X POST http://localhost:8080/debug/getToken/ | jq -r .token)
curl -s http://localhost:8080/labels/ -H "x-access-token: $TOKEN" | jq '.[0:3]'

kill $SERVER_PID
```

Expected: Labels show Type field with values (e.g., "protein", "course")

- [ ] **Step 4: Commit bootstrap.go update**

```bash
git add bootstrap.go
git commit -m "Update bootstrap.go to handle type column

Embedded bootstrap now creates type column and populates from CSV.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Chunk 5: Migration Script and Final Testing

### Task 8: Create Migration Script

**Files:**
- Create: `migration_add_label_type.sql`

- [ ] **Step 1: Write migration script**

Create `migration_add_label_type.sql` in project root:

```sql
-- Migration: Add type column to label table
-- Date: 2026-03-12
-- Purpose: Add label type/category support (meta-labels)

-- Add type column if it doesn't exist (idempotent check)
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'label'
  AND COLUMN_NAME = 'type';

SET @query = IF(@col_exists = 0,
    'ALTER TABLE label ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT ''''',
    'SELECT ''Column already exists'' AS msg');
PREPARE stmt FROM @query;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Populate types for existing labels
-- Proteins (8)
UPDATE label SET type = 'protein' WHERE label_id IN (1,2,3,4,5,31,37,45);

-- Courses (6)
UPDATE label SET type = 'course' WHERE label_id IN (8,9,12,17,35,36);

-- Cuisines (4)
UPDATE label SET type = 'cuisine' WHERE label_id IN (14,15,16,28);

-- Dietary (3)
UPDATE label SET type = 'dietary' WHERE label_id IN (6,7,26);

-- Dishes (15)
UPDATE label SET type = 'dish' WHERE label_id IN (10,11,13,18,20,21,22,23,24,27,32,33,34,42,44);

-- Attributes (3)
UPDATE label SET type = 'attribute' WHERE label_id IN (29,30,38);

-- Ingredients (2)
UPDATE label SET type = 'ingredient' WHERE label_id IN (19,25);

-- Preparation methods (2)
UPDATE label SET type = 'preparation' WHERE label_id IN (40,41);

-- IDs 39 (lamp), 43 (summer), 46 (eatsy) remain empty (unclear/typos)

-- Verification query (run after migration to confirm)
-- SELECT type, COUNT(*) as count FROM label GROUP BY type ORDER BY count DESC;
```

- [ ] **Step 2: Commit migration script**

```bash
git add migration_add_label_type.sql
git commit -m "Add migration script for label type column

Idempotent script adds type column and populates with category
values for existing labels.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 9: Final Verification

**Files:**
- All modified files

- [ ] **Step 1: Run complete test suite**

```bash
go test -v ./...
```

Expected: All tests pass

- [ ] **Step 2: Build application**

```bash
go build
```

Expected: No errors, binary created

- [ ] **Step 3: Manual end-to-end test**

```bash
# Start server with fresh bootstrap
./gorecipes -config mem.config -bootstrap &
SERVER_PID=$!
sleep 2

# Get token
TOKEN=$(curl -s -X POST http://localhost:8080/debug/getToken/ | jq -r .token)

# Fetch labels (should have types)
echo "=== Labels with types ==="
curl -s http://localhost:8080/labels/ -H "x-access-token: $TOKEN" | jq '.[0:5]'

# Update a label type
echo "=== Update label 1 type ==="
curl -v -X PUT http://localhost:8080/priv/label/id/1 \
  -H "x-access-token: $TOKEN" \
  -d "type=protein"

# Verify update
echo "=== Verify type update ==="
curl -s http://localhost:8080/labels/ -H "x-access-token: $TOKEN" | jq '.[] | select(.ID==1)'

# Update label name and type
echo "=== Update label 1 name and type ==="
curl -v -X PUT http://localhost:8080/priv/label/id/1 \
  -H "x-access-token: $TOKEN" \
  -d "label=poultry&type=protein"

# Verify name and type update
echo "=== Verify name and type update ==="
curl -s http://localhost:8080/labels/ -H "x-access-token: $TOKEN" | jq '.[] | select(.ID==1)'

# Test validation (should fail)
echo "=== Test type too long (should return 400) ==="
curl -v -X PUT http://localhost:8080/priv/label/id/1 \
  -H "x-access-token: $TOKEN" \
  -d "type=123456789012345678901"

# Cleanup
kill $SERVER_PID
```

Expected:
- Labels show Type field with values
- Update operations return 204
- Updates persist and are visible in subsequent fetches
- Type too long returns 400
- Type normalization to lowercase works

- [ ] **Step 4: Update CLAUDE.md documentation**

Update `CLAUDE.md` to reflect the Type field:

1. Find the "Label" structure documentation (around line 99-102) and update it:

```markdown
A "label" object has the following properties:
 - `ID` (int): the primary identifier for this label
 - `Label` (string): the label's name
 - `Icon` (string, optional): an emoji or character used as a visual icon for this label in the recipe list
 - `Type` (string, optional): category/type of this label (e.g., "protein", "course", "cuisine", "dietary")
```

2. Find the Database Model section for Label (around line 98-102) and verify it mentions:

```markdown
## Label
A `Label` has the following attributes:
 - `ID` (`label_id` in the db): the primary key for this label in the db
 - `Label`: the label's name
 - `Icon`: optional emoji/character icon
 - `Type`: optional category (protein, course, cuisine, dietary, dish, attribute, ingredient, preparation)
```

3. Commit the documentation update:

```bash
git add CLAUDE.md
git commit -m "Update CLAUDE.md to document Label type field

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Implementation Complete

All tasks completed. The label type feature is fully implemented with:

- Type field added to Label struct
- Type validation (max 20 chars, lowercase normalization)
- updateLabel model method accepts and validates type parameter
- editLabel handler accepts optional type form parameter
- Bootstrap data updated with type column
- Migration script for production database
- Comprehensive tests (unit and integration)

**Next Steps:**
1. Review all changes
2. Test manually with frontend (once frontend is updated)
3. Deploy migration script to production database
4. Deploy updated application to staging/production
5. Update TODO.md to mark this feature complete
