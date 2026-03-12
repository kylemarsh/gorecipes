# Label Icon Attribute Design

## Overview
Add an `icon` field to the Label database table and struct to hold a single character (emoji or other Unicode grapheme) to visually represent each label. Provide an API endpoint to update label names and icons.

## Goals
- Store an optional icon (emoji/character) for each label
- Validate icons as single grapheme clusters to ensure proper display
- Provide API endpoint to update existing labels (name and/or icon)
- Migrate production database with initial icon values
- Update bootstrapping data to include icons

## Non-Goals
- Adding icons to the existing label creation endpoint (PUT `/priv/label/{label_name}`)
- Supporting multi-character or image-based icons
- Client-side icon rendering (handled separately by frontend)

## Database Schema

### Label Table Changes
Add `icon` column to the existing `label` table:

```sql
ALTER TABLE label ADD COLUMN icon VARCHAR(255) NOT NULL DEFAULT '';
```

**Column Specifications:**
- Type: `VARCHAR(255)` (supports multi-byte UTF-8 characters)
- NOT NULL with DEFAULT '' (empty string indicates no icon)
- No uniqueness constraint (multiple labels can share icons)

### Updated Schema
```sql
CREATE TABLE label (
  label_id INT NOT NULL AUTO_INCREMENT,
  label VARCHAR(255) NOT NULL,
  icon VARCHAR(255) NOT NULL DEFAULT '',
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
}
```

### Model Methods

#### createLabel (existing - no changes)
Keep existing signature and behavior:
- Parameters: `labelName string`
- Icon defaults to empty string in database
- Returns created Label with empty Icon field

#### updateLabel (new)
Update existing label's name and/or icon:

```go
func updateLabel(labelID int, newName string, icon string) error
```

**Parameters:**
- `labelID` - ID of label to update
- `newName` - New label name (normalized to lowercase)
- `icon` - New icon value (validated as single grapheme or empty)

**Validation:**
1. Icon validation: Must be exactly 1 grapheme cluster or empty string
   - Use `github.com/rivo/uniseg.GraphemeClusterCount()`
   - Handles complex cases: emoji with modifiers (🌶️), country flags (🇲🇽), combining characters
   - Return error if count > 1
2. Name uniqueness: If newName differs from current name
   - Case-insensitive uniqueness check (exclude current label)
   - Query: `SELECT COUNT(*) FROM label WHERE LOWER(label) = ? AND label_id != ?`
   - Return error if conflict exists
3. Name normalization: Convert newName to lowercase using `strings.ToLower()`

**Behavior:**
- Return error if labelID doesn't exist
- UPDATE both label and icon fields
- Empty string for icon clears the icon (sets to '')

**Error Cases:**
- Label not found: return error (handler returns 404)
- Icon validation fails: return error (handler returns 400)
- Name conflict: return error (handler returns 409)

## API Changes

### New Endpoint: Update Label

**Route:** `PUT /priv/label/id/{label_id}`
**Handler:** `editLabel` in `privileged.go`
**Authentication:** Required (privRouter)

**Path Parameters:**
- `label_id` (int) - ID of label to update

**Form Parameters (both optional):**
- `label` (string) - New label name
- `icon` (string) - New icon value

**Behavior:**
1. Extract `label_id` from route, validate as integer
2. Extract optional `label` and `icon` form parameters
3. Fetch existing label by ID (return 404 if not found)
4. Determine final values:
   - If parameter missing, use existing value
   - If parameter provided, use new value
5. Call `updateLabel()` with final values
6. Return appropriate status code

**Response Codes:**
- `204 No Content` - Update successful
- `400 Bad Request` - Invalid label_id format or icon validation failed
- `404 Not Found` - Label doesn't exist
- `409 Conflict` - Label name conflicts with existing label

**Examples:**

Update icon only:
```
PUT /priv/label/id/14
icon=🌮
```

Update name only:
```
PUT /priv/label/id/14
label=mexican
```

Update both:
```
PUT /priv/label/id/14
label=mexican&icon=🇲🇽
```

Clear icon:
```
PUT /priv/label/id/14
icon=
```

### Existing Endpoint: No Changes

**Route:** `PUT /priv/label/{label_name}`
**Handler:** `addLabel` (unchanged)

Remains as get-or-create operation. Created labels have empty icon by default.

## Routing

Add to `privRouter` in `main.go`:

```go
privRouter.Handle("/label/id/{label_id}", wrappedHandler(editLabel)).Methods("PUT")
```

No conflict with existing route:
```go
privRouter.Handle("/label/{label_name}", wrappedHandler(addLabel)).Methods("PUT")
```

Routes are distinguished by `/label/id/` prefix vs `/label/` pattern.

## Bootstrapping Data

### Update bootstrapping/labels.csv
Add third column for icon values based on `production_labels.csv` mappings:

```csv
"label_id";"label";"icon"
"1";"chicken";"🐓"
"2";"beef";"🐄"
...
```

**Format:**
- Semicolon-delimited
- Three columns: label_id, label, icon
- Empty string for labels without icons
- Quoted values

### Update bootstrap_recipes.go
Modify CSV parsing in `bootstrapping/bootstrap_recipes.go`:

1. Update label CSV reader to expect 3 columns
2. Parse icon column (third field)
3. Update INSERT statement:
   ```go
   INSERT INTO label (label_id, label, icon) VALUES (?, ?, ?)
   ```
4. Include icon value in Exec parameters

### Update bootstrap.go
Modify embedded CSV logic in `bootstrap.go`:

1. Update hardcoded CSV strings to include icon column
2. Parse 3 fields per line instead of 2
3. Update INSERT statement to include icon column
4. Include icon value in Exec parameters

**Note:** Both bootstrap programs must be updated to maintain consistency between in-memory and file-based bootstrapping.

## Migration Script

Create `migration_add_label_icon.sql` for production database:

```sql
-- Add icon column if it doesn't exist
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'label'
  AND COLUMN_NAME = 'icon';

SET @query = IF(@col_exists = 0,
    'ALTER TABLE label ADD COLUMN icon VARCHAR(255) NOT NULL DEFAULT ''''',
    'SELECT ''Column already exists'' AS msg');
PREPARE stmt FROM @query;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Populate icons for existing labels
UPDATE label SET icon = '🐄' WHERE label_id = 1;  -- Beef
UPDATE label SET icon = '🐓' WHERE label_id = 2;  -- Chicken
UPDATE label SET icon = '🐖' WHERE label_id = 3;  -- Pork
UPDATE label SET icon = '🐟' WHERE label_id = 4;  -- Fish
UPDATE label SET icon = '🐑' WHERE label_id = 5;  -- Lamb
UPDATE label SET icon = '🥦' WHERE label_id = 6;  -- Vegetarian
UPDATE label SET icon = 'Ⓥ' WHERE label_id = 7;  -- Vegan
UPDATE label SET icon = '🍳' WHERE label_id = 8;  -- Breakfast
UPDATE label SET icon = '🥤' WHERE label_id = 9;  -- Drink
UPDATE label SET icon = '🍜' WHERE label_id = 10; -- SoupStew
UPDATE label SET icon = '🥬' WHERE label_id = 11; -- Salad
UPDATE label SET icon = '🥟' WHERE label_id = 12; -- Appetizer
UPDATE label SET icon = '🥪' WHERE label_id = 13; -- Sandwich
UPDATE label SET icon = '🇲🇽' WHERE label_id = 14; -- Mexican
UPDATE label SET icon = '🥢' WHERE label_id = 15; -- Asian
-- MiddleEast (16) - no icon
UPDATE label SET icon = '🍦' WHERE label_id = 17; -- Dessert
UPDATE label SET icon = '🍞' WHERE label_id = 18; -- Bread
UPDATE label SET icon = '🥕' WHERE label_id = 19; -- Vegetable
UPDATE label SET icon = '🍪' WHERE label_id = 20; -- Cookie
UPDATE label SET icon = '🎂' WHERE label_id = 21; -- Cake
UPDATE label SET icon = '🍬' WHERE label_id = 22; -- Candy
-- Cheesecake (23) - no icon
-- CreamCustard (24) - no icon
UPDATE label SET icon = '🍏' WHERE label_id = 25; -- Fruit
UPDATE label SET icon = 'Ⓖ' WHERE label_id = 26; -- GlutenFree
UPDATE label SET icon = '🍝' WHERE label_id = 27; -- Pasta
UPDATE label SET icon = '🇬🇷' WHERE label_id = 28; -- Greek
UPDATE label SET icon = '🌶️' WHERE label_id = 29; -- Spicy
UPDATE label SET icon = '⚡' WHERE label_id = 30; -- Quick
UPDATE label SET icon = '🦐' WHERE label_id = 31; -- Shrimp
-- Sauce (32) - no icon
UPDATE label SET icon = '🍚' WHERE label_id = 33; -- Rice
-- StarchSide (34) - no icon
-- Side (35) - no icon
UPDATE label SET icon = '🍽️' WHERE label_id = 36; -- Main
UPDATE label SET icon = '🦃' WHERE label_id = 37; -- Turkey
UPDATE label SET icon = '⏰' WHERE label_id = 38; -- Batch
-- lamp (39) - no icon (typo/duplicate)
UPDATE label SET icon = '🌡️' WHERE label_id = 40; -- sousvide
-- air fryer (41) - no icon
UPDATE label SET icon = '🥣' WHERE label_id = 42; -- soup
-- summer (43) - no icon
-- sauces (44) - no icon (duplicate)
-- tofu (45) - no icon
-- eatsy (46) - no icon
```

**Script Properties:**
- Idempotent: Checks if column exists before adding
- Explicit mappings: Each label ID updated individually
- Empty icons: Labels without icons remain as empty string (default)

## Dependencies

Add to `go.mod`:
```
github.com/rivo/uniseg v0.4.7
```

**Why this library:**
- Proper Unicode grapheme cluster counting
- Handles complex cases: emoji with modifiers (🌶️ = 2 code points, 1 grapheme), country flags (🇲🇽 = 2 code points, 1 grapheme)
- Well-maintained, small footprint
- Standard solution for Go grapheme handling

## Validation Logic

### Icon Validation
Use `github.com/rivo/uniseg.GraphemeClusterCount()`:

```go
func validateIcon(icon string) error {
    if icon == "" {
        return nil // Empty is valid
    }

    count := uniseg.GraphemeClusterCount(icon)
    if count != 1 {
        return fmt.Errorf("icon must be exactly 1 character, got %d", count)
    }
    return nil
}
```

**Test Cases:**
- Empty string: valid
- Single ASCII: `"G"` → valid
- Single emoji: `"🐓"` → valid
- Emoji with modifier: `"🌶️"` → valid (1 grapheme, 2+ code points)
- Country flag: `"🇲🇽"` → valid (1 grapheme, 2 code points)
- Multiple emojis: `"🐓🐄"` → invalid
- Combining characters: Should count as 1 grapheme

### Label Name Validation
Existing uniqueness validation enhanced:

```go
func validateLabelName(name string, excludeID int) error {
    normalized := strings.ToLower(name)

    var count int
    q := "SELECT COUNT(*) FROM label WHERE LOWER(label) = ? AND label_id != ?"
    err := db.Get(&count, q, normalized, excludeID)
    if err != nil {
        return err
    }

    if count > 0 {
        return fmt.Errorf("label name already exists: %s", name)
    }
    return nil
}
```

## Testing Strategy

### Unit Tests (model_test.go if exists)
- Icon validation: empty, single grapheme, multiple graphemes
- Complex graphemes: emoji with modifiers, country flags, combining chars
- Label uniqueness: case-insensitive conflicts, self-exclusion
- Name normalization: uppercase → lowercase

### Integration Tests
- Update icon only: verify icon changes, name unchanged
- Update name only: verify name changes (lowercase), icon unchanged
- Update both: verify both fields update
- Clear icon: empty string sets icon to ''
- Name conflict: returns error
- Invalid icon: returns error
- Nonexistent label: returns 404
- Partial updates: missing parameters use existing values

### Manual Test Cases
1. Create label via existing endpoint, verify empty icon
2. Update icon via new endpoint, verify in database
3. Update name, verify lowercase normalization
4. Attempt duplicate name, verify 409 error
5. Clear icon with empty string, verify in database
6. Complex emoji icons (flags, modifiers), verify storage and retrieval

## Error Handling

### Model Layer Errors
- Icon validation failure: Return descriptive error
- Name conflict: Return descriptive error
- Label not found: Return `sql.ErrNoRows`
- Database errors: Return raw error for handler to interpret

### Handler Layer Response Codes
| Error Condition | Status Code | Response Body |
|----------------|-------------|---------------|
| Invalid label_id format | 400 | "label ID must be an integer" |
| Icon validation failed | 400 | "icon must be exactly 1 character" |
| Label not found | 404 | "label does not exist" |
| Name conflict | 409 | "label name already exists" |
| Database error | 500 | "problem updating label" |
| Success | 204 | (empty) |

## Implementation Notes

### Code Organization
- Model changes: `model.go`
- Handler changes: `privileged.go`
- Route registration: `main.go`
- Bootstrap updates: `bootstrap.go` and `bootstrapping/bootstrap_recipes.go`
- Migration script: `migration_add_label_icon.sql` (root directory)

### Consistency Patterns
Follow existing patterns from recipe `new` flag implementation:
- Handler structure matches `flagNote`/`unFlagNote`
- Model method matches `setRecipeNewFlag`
- Form parameter extraction via `r.FormValue()`
- Error handling via `appError` struct
- Response codes: 204 success, 400/404/409/500 errors

### CSV Format Consistency
Both bootstrap files use same CSV format:
- Semicolon delimiter (`;`)
- Quoted values (`"value"`)
- Header row with column names
- Icon column third position

### Database Normalization
Label names stored lowercase for consistency:
- `addLabel`: Already lowercases via `strings.ToLower(mux.Vars(r)["label_name"])`
- `editLabel`: Must lowercase new name before storing
- Queries: Use LOWER() for case-insensitive comparisons

## Future Considerations

### Out of Scope (for future work)
- Adding icon parameter to existing addLabel endpoint
- Bulk icon updates
- Icon validation on frontend
- Icon search/filter functionality
- Icon suggestions/autocomplete
- Label type attribute (separate TODO item)

### Migration Path
If label creation needs icon support later:
1. Add optional icon parameter to `addLabel` handler
2. Pass to `createLabel` (requires signature change)
3. Update existing clients to send icon on creation
4. Maintains backward compatibility (icon defaults to empty)

## Acceptance Criteria

- [ ] Label struct includes Icon field
- [ ] Database schema includes icon column with proper type and default
- [ ] updateLabel model method validates and updates both fields
- [ ] editLabel handler accepts and processes form parameters
- [ ] Route registered at PUT /priv/label/id/{label_id}
- [ ] Icon validation uses grapheme cluster counting
- [ ] Name validation enforces case-insensitive uniqueness
- [ ] Name normalization converts to lowercase
- [ ] Empty string clears icon
- [ ] Missing parameters preserve existing values
- [ ] Proper error codes for all failure cases
- [ ] bootstrapping/labels.csv includes icon column
- [ ] bootstrap_recipes.go reads and inserts icon
- [ ] bootstrap.go reads and inserts icon
- [ ] Migration script is idempotent
- [ ] Migration script populates production icons
- [ ] uniseg dependency added to go.mod
- [ ] Tests cover validation edge cases
- [ ] Tests cover partial update scenarios
