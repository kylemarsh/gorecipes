# Label Type Attribute (Backend) Design

## Overview
Add a `type` field to labels to categorize them (e.g., "protein", "course", "cuisine", "dietary"). This enables the frontend to group and filter labels by category dynamically.

## Goals
- Store an optional type/category for each label
- Validate types as lowercase strings with max 20 character length
- Provide API endpoint to update label type (via existing editLabel endpoint)
- Migrate production database with initial type values
- Update bootstrapping data to include types

## Non-Goals
- Creating separate endpoint for type-only updates (use existing editLabel)
- Adding type parameter to addLabel endpoint (labels created without type, can be set later)
- Type suggestions or autocomplete

## Database Schema

### Label Table Changes
Add `type` column to the existing `label` table:

```sql
ALTER TABLE label ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT '';
```

**Column Specifications:**
- Type: `VARCHAR(20)` (max 20 characters, enforced in application)
- NOT NULL with DEFAULT '' (empty string indicates uncategorized)
- No uniqueness constraint (multiple labels can share types)
- Lowercase only (enforced in application layer)

### Updated Schema
```sql
CREATE TABLE label (
  label_id INT NOT NULL AUTO_INCREMENT,
  label VARCHAR(255) NOT NULL,
  icon VARCHAR(255) NOT NULL DEFAULT '',
  type VARCHAR(20) NOT NULL DEFAULT '',
  PRIMARY KEY (label_id),
  KEY label (label)
);
```

## Data Model

### Label Struct
Update `model.go` Label struct:

```go
type Label struct {
    ID    int    `db:"label_id"`
    Label string
    Icon  string
    Type  string
}
```

### Model Methods

#### createLabel (existing - no changes)
Keep existing signature and behavior:
- Parameters: `labelName string`
- Type defaults to empty string in database
- Returns created Label with empty Type field

#### updateLabel (existing - extend signature)
Extend existing method to accept type parameter:

```go
func updateLabel(labelID int, newName string, icon string, labelType string) error
```

**Parameters:**
- `labelID` - ID of label to update
- `newName` - New label name (normalized to lowercase)
- `icon` - New icon value (validated as single grapheme or empty)
- `labelType` - New type value (validated and normalized to lowercase)

**Type Validation:**
1. Empty string is valid (means uncategorized)
2. Max 20 characters - return error if length > 20
3. Lowercase normalization using `strings.ToLower()`

**Behavior:**
- Return error if labelID doesn't exist
- UPDATE all fields (label, icon, type) atomically
- Empty string for type clears the type (sets to '')
- Existing name and icon validation remains unchanged

**Error Cases:**
- Label not found: return error (handler returns 404)
- Icon validation fails: return error (handler returns 400)
- Name conflict: return error (handler returns 409)
- Type too long: return error (handler returns 400)

## API Changes

### Modified Endpoint: Update Label

**Route:** `PUT /priv/label/id/{label_id}`
**Handler:** `editLabel` in `privileged.go` (existing, extend)
**Authentication:** Required (privRouter)

**Path Parameters:**
- `label_id` (int) - ID of label to update

**Form Parameters (all optional):**
- `label` (string) - New label name
- `icon` (string) - New icon value
- `type` (string) - New type value

**Behavior:**
1. Extract `label_id` from route, validate as integer
2. Parse form data
3. Extract optional `label`, `icon`, and `type` form parameters
4. Fetch existing label by ID (return 404 if not found)
5. Determine final values:
   - If parameter not provided (not in form), use existing value
   - If parameter provided (even empty string), use new value
6. Call `updateLabel()` with final values
7. Return appropriate status code

**Response Codes:**
- `204 No Content` - Update successful
- `400 Bad Request` - Invalid label_id format, icon validation failed, or type too long
- `404 Not Found` - Label doesn't exist
- `409 Conflict` - Label name conflicts with existing label

**Examples:**

Update type only:
```
PUT /priv/label/id/14
type=cuisine
```

Update name and type:
```
PUT /priv/label/id/14
label=mexican&type=cuisine
```

Update all three fields:
```
PUT /priv/label/id/14
label=mexican&icon=🇲🇽&type=cuisine
```

Clear type:
```
PUT /priv/label/id/14
type=
```

### Existing Endpoint: No Changes

**Route:** `PUT /priv/label/{label_name}`
**Handler:** `addLabel` (unchanged)

Remains as get-or-create operation. Created labels have empty type by default.

## Bootstrapping Data

### Label Type Categories

**Type Assignments:**
- **protein**: beef, chicken, pork, fish, lamb, shrimp, turkey, tofu (8 labels)
- **course**: main, dessert, breakfast, appetizer, side, drink (6 labels)
- **cuisine**: mexican, asian, middleeast, greek (4 labels)
- **dietary**: vegetarian, vegan, glutenfree (3 labels)
- **dish**: soupstew, soup, salad, sandwich, bread, pasta, cookie, cake, candy, cheesecake, creamcustard, sauce, sauces, rice, starchside (15 labels)
- **attribute**: spicy, quick, batch (3 labels)
- **ingredient**: fruit, vegetable (2 labels)
- **preparation**: sousvide, air fryer (2 labels)
- **empty**: summer, eatsy, lamp (3 labels - unclear/typos)

### Update bootstrapping/labels.csv
Add fourth column for type values:

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

**Format:**
- Semicolon-delimited
- Four columns: label_id, label, icon, type
- Empty string for labels without types
- Quoted values

### Update bootstrap_recipes.go
Modify CSV parsing in `bootstrapping/bootstrap_recipes.go`:

Update label table definition in info map (lines 46-52):
- Update `create_mysql`: add `type VARCHAR(20) NOT NULL DEFAULT ''` column
- Update `create_sqlite3`: add `type VARCHAR(20) NOT NULL DEFAULT ''` column
- Update `insert`: `INSERT INTO label (label_id, label, icon, type) VALUES (?, ?, ?, ?)`

CSV parsing automatically handles 4 columns (no code change needed).

### Update bootstrap.go
Modify embedded CSV logic in `bootstrap.go`:

Update label table definition in info map (lines 29-35):
- Update `create_mysql`: add `type VARCHAR(20) NOT NULL DEFAULT ''` column
- Update `create_sqlite3`: add `type VARCHAR(20) NOT NULL DEFAULT ''` column
- Update `insert`: `INSERT INTO label (label_id, label, icon, type) VALUES (?, ?, ?, ?)`

CSV parsing automatically handles 4 columns (no code change needed).

**Note:** Both bootstrap programs must be updated to maintain consistency.

## Migration Script

Create `migration_add_label_type.sql` for production database:

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

**Script Properties:**
- Idempotent: Checks if column exists before adding
- Explicit mappings: Each label ID updated individually
- Empty types: Labels without clear categories remain as empty string

## Validation Logic

### Type Validation
Add validation function in `util.go`:

```go
func validateType(labelType string) error {
    if labelType == "" {
        return nil // Empty is valid
    }

    if len(labelType) > 20 {
        return fmt.Errorf("type must be 20 characters or less, got %d", len(labelType))
    }
    return nil
}
```

### Model Updates
Update `updateLabel` in `model.go`:

```go
func updateLabel(labelID int, newName string, icon string, labelType string) error {
    // Validate icon (existing)
    if err := validateIcon(icon); err != nil {
        return err
    }

    // Validate type (new)
    if err := validateType(labelType); err != nil {
        return err
    }

    // Normalize type to lowercase
    normalizedType := strings.ToLower(labelType)

    // Fetch existing label to check if it exists
    existing, err := labelByID(labelID)
    if err != nil {
        return err // Returns sql.ErrNoRows if not found
    }

    // Normalize new name to lowercase
    normalizedName := strings.ToLower(newName)

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
            return fmt.Errorf("label name already exists: %s", newName)
        }
    }

    // Update all three fields
    q := "UPDATE label SET label = ?, icon = ?, type = ? WHERE label_id = ?"
    connect()
    _, err = db.Exec(q, normalizedName, icon, normalizedType, labelID)
    return err
}
```

### Handler Updates
Update `editLabel` in `privileged.go`:

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
        if strings.Contains(err.Error(), "icon must be") {
            return &appError{http.StatusBadRequest, err.Error(), err}
        }
        if strings.Contains(err.Error(), "type must be") {
            return &appError{http.StatusBadRequest, err.Error(), err}
        }
        if strings.Contains(err.Error(), "already exists") {
            return &appError{http.StatusConflict, err.Error(), err}
        }
        return &appError{http.StatusInternalServerError, "problem updating label", err}
    }

    w.WriteHeader(http.StatusNoContent)
    return nil
}
```

## Testing Strategy

### Unit Tests

**Type validation (util_test.go):**
- Empty string: valid
- 20 characters: valid (boundary)
- 21 characters: invalid
- Lowercase string: valid
- Mixed case string: valid (normalized to lowercase)

**updateLabel method (model_test.go):**
- Update type only: verify type changes, name/icon unchanged
- Update all fields: verify all update
- Type normalization: uppercase -> lowercase
- Empty type: clears type field
- Type too long: returns error
- Nonexistent label: returns error

### Integration Tests

**editLabel handler (privileged_test.go):**
- Update type via form parameter
- Partial update: type only
- Clear type with empty string
- Missing type parameter: preserves existing type
- Type validation error: returns 400
- Type normalization: verify lowercase storage

### Manual Testing
1. Bootstrap database, verify labels have types
2. Update label type via API, verify in database
3. Create label via addLabel, verify type is empty
4. Update type to invalid value (too long), verify error
5. Clear type with empty string, verify in database

## Error Handling

### Model Layer Errors
- Type validation failure: Return descriptive error with length
- Icon validation failure: Return existing error
- Name conflict: Return existing error
- Label not found: Return `sql.ErrNoRows`
- Database errors: Return raw error for handler to interpret

### Handler Layer Response Codes
| Error Condition | Status Code | Response Body |
|----------------|-------------|---------------|
| Invalid label_id format | 400 | "label ID must be an integer" |
| Type too long | 400 | "type must be 20 characters or less" |
| Icon validation failed | 400 | "icon must be exactly 1 character" |
| Label not found | 404 | "label does not exist" |
| Name conflict | 409 | "label name already exists" |
| Database error | 500 | "problem updating label" |
| Success | 204 | (empty) |

## Implementation Notes

### Code Organization
- Model changes: `model.go` (Label struct, updateLabel method)
- Handler changes: `privileged.go` (editLabel handler)
- Utility: `util.go` (validateType function)
- Routing: `main.go` (no changes - uses existing route)
- Bootstrap updates: `bootstrap.go` and `bootstrapping/bootstrap_recipes.go`
- Migration script: `migration_add_label_type.sql` (root directory)

### Consistency Patterns
- Handler structure matches existing `editLabel` implementation
- Model method extends existing `updateLabel` signature
- Validation follows `validateIcon` pattern
- Form parameter extraction via `r.FormValue()` and `r.Form.Has()`
- Error handling via `appError` struct
- Response codes: 204 success, 400/404/409/500 errors

### CSV Format
- Semicolon delimiter (`;`)
- Quoted values (`"value"`)
- Header row with column names
- Type column fourth position

### Database Normalization
- Label names: stored lowercase (existing)
- Label types: stored lowercase (new)
- Queries: use LOWER() for case-insensitive comparisons (existing pattern)

## Future Considerations

### Out of Scope
- Adding type parameter to addLabel endpoint
- UI dropdown for selecting types
- Renaming types (would require updating all labels with that type)

## Acceptance Criteria

- [ ] Label struct includes Type field
- [ ] Database schema includes type column with VARCHAR(20) and default ''
- [ ] updateLabel model method accepts and validates type parameter
- [ ] Type validation enforces max 20 chars and normalizes to lowercase
- [ ] editLabel handler accepts optional type form parameter
- [ ] Missing type parameter preserves existing value
- [ ] Empty string type parameter clears type
- [ ] Proper error codes for validation failures
- [ ] bootstrapping/labels.csv includes type column with all labels categorized
- [ ] bootstrap_recipes.go parses and inserts type column
- [ ] bootstrap.go parses and inserts type column
- [ ] Migration script is idempotent
- [ ] Migration script populates types for all production labels
- [ ] Tests cover type validation edge cases
- [ ] Tests cover partial update scenarios with type
