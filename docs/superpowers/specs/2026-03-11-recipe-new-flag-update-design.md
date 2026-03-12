# Recipe "New" Flag Update Endpoint Support

**Date:** 2026-03-11
**Status:** Approved

## Overview

Add support for updating the recipe "new" flag through the existing `PUT /priv/recipe/{id}` endpoint. This allows users to update recipe content and the new flag in a single request, complementing the existing dedicated `mark_cooked` and `mark_new` endpoints.

## Background

The "new" flag was recently added to track recipes that haven't been cooked yet. Currently:
- The flag exists in the Recipe struct and database schema
- Dedicated endpoints exist: `PUT /priv/recipe/{id}/mark_cooked` and `PUT /priv/recipe/{id}/mark_new`
- The main recipe update endpoint (`PUT /priv/recipe/{id}`) doesn't support updating this flag
- Frontend uses an HTML checkbox to control the flag state

## Design Decision

**Approach: Checkbox with explicit update**

The "new" field will be treated as a standard HTML checkbox:
- When checked, the browser sends `new=on`
- When unchecked, the field is absent from form data
- Handler interprets any non-empty value as `true`, empty/absent as `false`
- Every update explicitly sets the new flag based on checkbox state

This matches standard HTML form behavior and keeps the implementation simple and predictable.

## Components

### 1. Handler Function (`privileged.go:updateExistingRecipe`)

**Changes:**
- Add parsing of "new" form value after parsing other fields
- Convert to boolean: `isNew := r.FormValue("new") != ""`
- Pass boolean to `updateRecipe()` function call

**Code location:** `privileged.go:106-131`

### 2. Model Function (`model.go:updateRecipe`)

**Changes:**
- Add `isNew bool` parameter to function signature
- Add `new = ?` to the UPDATE statement
- Add `isNew` to the Exec parameters

**Current signature:**
```go
func updateRecipe(recipeId int, title string, body string, activeTime int, totalTime int) error
```

**New signature:**
```go
func updateRecipe(recipeId int, title string, body string, activeTime int, totalTime int, isNew bool) error
```

**Code location:** `model.go:220-230`

### 3. Routes

No changes required. The existing route already handles this endpoint:
```go
privRouter.Handle("/recipe/{id}", wrappedHandler(updateExistingRecipe)).Methods("PUT")
```

## Data Flow

1. **HTTP Request**: `PUT /priv/recipe/{id}` with form data
   - Required: `title`, `activeTime`, `totalTime`, `body`
   - Optional: `new` (checkbox sends "on" when checked, nothing when unchecked)

2. **Handler**: Parse all form values including "new", convert to boolean

3. **Model**: Execute SQL UPDATE including the `new` field

4. **Response**:
   - 204 No Content on success
   - 400 Bad Request for validation errors
   - 500 Internal Server Error for database errors

## Error Handling

**Validation:**
- "new" field requires no validation (presence = true, absence = false)
- Existing validations unchanged: recipe ID integer, title required, times integer

**Edge Cases:**
- Recipe doesn't exist: UPDATE affects 0 rows but doesn't error (standard SQL behavior)
- Malformed "new" value (e.g., "new=invalid"): Treated as true (non-empty = true)
- Multiple "new" values: `FormValue` returns first value (standard Go behavior)
- Missing "new" parameter: Treated as false (backward compatible)

**Database:**
- The `new` column already exists in schema
- No migration needed

## Testing

**Manual Testing:**
- Update recipe with checkbox checked → verify `new=true`
- Update recipe with checkbox unchecked → verify `new=false`
- Update recipe, toggle checkbox state → verify change persists
- Update without "new" parameter → verify sets to false

**Integration Tests:**
- Add test cases following the pattern from commits a529759, e5912df
- Test update with new=true
- Test update with new=false (or absent)
- Verify other fields update correctly alongside new flag

**Test Cases:**
- Form value "new=on" → `new=true`
- Form value "new=1" → `new=true`
- Form value "new=" → `new=false`
- No "new" field → `new=false`
- Update multiple fields including new flag

## Backward Compatibility

**Frontend Compatibility:**
- If frontend doesn't send "new" parameter, it defaults to false
- No breaking changes to existing API contract
- Dedicated `mark_cooked`/`mark_new` endpoints remain unchanged

**Database Compatibility:**
- Uses existing `new` column
- No schema changes required

## Implementation Notes

- Follow the pattern of existing update handlers (e.g., `updateExistingRecipe`)
- Use standard Go form parsing: `r.FormValue("new")`
- Boolean conversion: non-empty string = true, empty/absent = false
- Maintain consistent error handling with other handlers
- Use existing `setRecipeNewFlag()` pattern as reference for SQL statement

## Success Criteria

- Recipe updates can include the new flag
- Checkbox state correctly reflects in database after update
- All existing update functionality continues to work
- Integration tests pass
- Manual testing confirms checkbox behavior
