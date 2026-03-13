# TestBootstrap Failure - Label Count Mismatch

**Date:** 2026-03-12

## Bug

The `TestBootstrap` test was failing with:
```
Got 46 labels, expected 37
Got 46 labels, expected 38
Got 46 labels, expected 37
```

All other tests passed, but `TestBootstrap` failed with exit code 1.

## Root Cause

Commit `fe9291a` ("update bootstrapping data") added 9 new labels to `bootstrapping/labels.csv`, increasing the total label count from 37 to 46. The test expectations in `model_test.go` were not updated to reflect this change.

Additionally, the test was attempting to insert a label with ID 40 to test the bootstrap skip logic, but label ID 40 now exists in the CSV file (label "sousvide"), causing a duplicate key constraint violation.

## Fix

Updated `model_test.go` lines 39-47:

1. **Updated expected label count after initial bootstrap:** Changed from `37` to `46`
2. **Changed manual insert ID:** Changed from `40` to `50` (an unused ID)
3. **Updated expected count after manual insert:** Changed from `38` to `47`
4. **Updated expected count after force re-bootstrap:** Changed from `37` to `46`

```go
// Before:
checkDb(t, 0, 0, 0)
bootstrap(false)
checkDb(t, 37, 13, 32)

db.Exec("insert into label values (40, 'florp', '', '')")
bootstrap(false)
checkDb(t, 38, 13, 32)
bootstrap(true)
checkDb(t, 37, 13, 32)

// After:
checkDb(t, 0, 0, 0)
bootstrap(false)
checkDb(t, 46, 13, 32)

db.Exec("insert into label values (50, 'florp', '', '')")
bootstrap(false)
checkDb(t, 47, 13, 32)
bootstrap(true)
checkDb(t, 46, 13, 32)
```

## Verification

After the fix, all tests pass:
```
go test -v
PASS
ok  	github.com/kylemarsh/gorecipes	0.621s
```

## Prevention

When adding or removing records from bootstrap CSV files in `bootstrapping/`, always update the corresponding test expectations in `model_test.go`. The `TestBootstrap` function verifies counts for labels, recipes, and recipe_label junction table entries.
