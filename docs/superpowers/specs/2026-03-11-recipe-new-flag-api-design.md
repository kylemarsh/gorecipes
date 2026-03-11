# Recipe "New" Flag API Design

## Overview

This feature adds API endpoints to expose the existing `new` field on recipes, allowing clients to mark recipes as cooked (setting `new = false`) or mark them as new/uncooked (setting `new = true`). The implementation follows the established pattern used for note flagging in the codebase.

## Background

The Recipe struct in `model.go` already has a `New bool` field (line 35), and the database schema supports it (present in `bootstrapping/recipes.csv`). However, there are currently no API endpoints to modify this field. This design adds that capability.

## Architecture

The feature adds two new authenticated endpoints under the `/priv/recipe/{id}/` path:
- `PUT /priv/recipe/{id}/mark_cooked` - marks a recipe as cooked (sets `new = false`)
- `PUT /priv/recipe/{id}/mark_new` - marks a recipe as new/uncooked (sets `new = true`)

Both endpoints require authentication and follow the existing pattern used for note flagging.

## Components

### 1. Model Layer (`model.go`)

Add a new database function:

```go
func setRecipeNewFlag(recipeID int, isNew bool) error
```

**Behavior:**
- Executes an UPDATE query to set the `new` field on the specified recipe
- Returns error if database operation fails
- Follows the same pattern as `setNoteFlag()` at line 232

**Implementation pattern:**
```sql
UPDATE recipe SET new = ? WHERE recipe_id = ?
```

### 2. Handler Layer (`privileged.go`)

Add two new handler functions:

#### `flagRecipeCooked(w http.ResponseWriter, r *http.Request) *appError`

**Behavior:**
1. Extract recipe ID from URL parameter
2. Validate ID is an integer (return 400 Bad Request if not)
3. Check that recipe exists via `recipeByID()` (return 404 Not Found if missing)
4. Call `setRecipeNewFlag(recipeID, false)` to mark as cooked
5. Return 204 No Content on success

#### `unFlagRecipeCooked(w http.ResponseWriter, r *http.Request) *appError`

**Behavior:**
1. Extract recipe ID from URL parameter
2. Validate ID is an integer (return 400 Bad Request if not)
3. Check that recipe exists via `recipeByID()` (return 404 Not Found if missing)
4. Call `setRecipeNewFlag(recipeID, true)` to mark as new
5. Return 204 No Content on success

Both handlers follow the structure of `flagNote()` and `unFlagNote()` at lines 133-163.

### 3. Routing (`main.go`)

Add two new PUT routes to the `privRouter` (around line 48-49):

```go
privRouter.Handle("/recipe/{id}/mark_cooked", wrappedHandler(flagRecipeCooked)).Methods("PUT")
privRouter.Handle("/recipe/{id}/mark_new", wrappedHandler(unFlagRecipeCooked)).Methods("PUT")
```

These routes require authentication via the `authRequired` middleware.

### 4. Bootstrap Data (`bootstrapping/recipes.csv`)

Update the CSV to set `new = 1` for approximately 3-4 recipes to demonstrate the feature. Example selections:
- Recipe 2 (Butternut Squash and Sage Wontons)
- Recipe 5 (Ground Turkey Laap)
- Recipe 8 (Mushrooms Stuffed with Prosciutto)
- Recipe 11 (Pot Stickers Veggie)

This provides variety across different recipe types.

## Data Flow

### Marking a Recipe as Cooked

```
Client → PUT /priv/recipe/{id}/mark_cooked (with auth token)
    ↓
authRequired middleware validates token
    ↓
flagRecipeCooked handler:
  1. Extract recipe ID from URL
  2. Validate ID is integer
  3. Verify recipe exists (recipeByID)
  4. Call setRecipeNewFlag(recipeID, false)
    ↓
Database: UPDATE recipe SET new = 0 WHERE recipe_id = ?
    ↓
← 204 No Content
```

### Marking a Recipe as New

```
Client → PUT /priv/recipe/{id}/mark_new (with auth token)
    ↓
authRequired middleware validates token
    ↓
unFlagRecipeCooked handler:
  1. Extract recipe ID from URL
  2. Validate ID is integer
  3. Verify recipe exists (recipeByID)
  4. Call setRecipeNewFlag(recipeID, true)
    ↓
Database: UPDATE recipe SET new = 1 WHERE recipe_id = ?
    ↓
← 204 No Content
```

## Error Handling

### Authentication Errors (handled by middleware)
- **Missing/expired token**: 401 Unauthorized
- **Malformed token**: 400 Bad Request
- Handler never executes in these cases

### Invalid Recipe ID
- **Non-integer ID** (e.g., `/recipe/abc/mark_cooked`):
  - `strconv.Atoi()` fails
  - Return: 400 Bad Request, "recipe ID must be an integer"

- **Recipe doesn't exist** (e.g., `/recipe/999/mark_cooked`):
  - `recipeByID()` returns `sql.ErrNoRows`
  - Return: 404 Not Found, "recipe does not exist"

### Database Errors
- **Error checking if recipe exists**:
  - Return: 500 Internal Server Error, "Problem loading recipe"

- **Error updating the flag**:
  - Return: 500 Internal Server Error, "problem setting recipe new flag"

All error patterns match existing handlers (`flagNote`, `unFlagNote`, etc.).

## API Examples

### Mark Recipe as Cooked
```bash
curl -X PUT \
  -H "x-access-token: $TOKEN" \
  http://localhost:8080/priv/recipe/5/mark_cooked
```

**Success Response:** 204 No Content

### Mark Recipe as New
```bash
curl -X PUT \
  -H "x-access-token: $TOKEN" \
  http://localhost:8080/priv/recipe/5/mark_new
```

**Success Response:** 204 No Content

## Testing Considerations

- Verify routes are registered under `privRouter` (ensures `authRequired` middleware is applied)
- Test with valid recipe IDs (expect 204 No Content)
- Test with non-existent recipe IDs (expect 404 Not Found)
- Test with non-integer IDs (expect 400 Bad Request)
- Verify database state changes after each operation (query recipe to confirm `new` field updated)
- Test toggling the same recipe multiple times (cooked → new → cooked)
- Verify the `new` field is properly returned in GET recipe responses
