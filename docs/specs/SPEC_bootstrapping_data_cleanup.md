# Spec: Bootstrapping Data Cleanup

## Objective
Update the CSV bootstrapping data to provide comprehensive test coverage for all database features and edge cases.

## Current State

### Existing Data Problems
1. **Label distribution**: 12 of 13 recipes tagged as "appetizer", poor course variety
2. **Recipe flags**: All recipes have `deleted=false`, limited `new` flag variety
3. **Time fields**: All recipes have `0` for both time fields
4. **Label types**: Some labels have empty types (intentional for testing)
5. **Notes**: Only 3 notes across 2 recipes, limited coverage

### Existing Data to Preserve
- **Users**: foo (admin), koko, ashai (non-admin) - keep as is
- **Labels**: Keep all existing labels, including intentional oddities (lamp typo, eatsy, summer without types)
- **Recipes to keep**:
  - Recipe 2: Butternut Squash Wontons (longest)
  - Recipe 10: Steamed Pork Buns (second longest)
  - 1-2 existing appetizers
- **Notes**: Keep existing 3 notes, add more

## Desired Outcome

### Recipe Distribution (20 total)

**By Course Type:**
- Main: 4 recipes
- Dessert: 4 recipes
- Drink: 2 recipes
- Appetizer: 3 recipes (mostly from existing set)
- Breakfast: 2 recipes
- Side: 3 recipes
- No course label: 2 recipes (sauce, spice mix)

**Recipe Details:**
- Mix of real recipes (2-4 kept from existing) and test stubs (16-18 new)
- Times rounded to nearest 10 minutes
- Realistic time combinations

**Flag States:**
- `new=true`: 5 recipes distributed across courses
- `deleted=true`: 2 recipes (one appetizer, one main)
- Rest: `new=false`, `deleted=false`

**Time Distribution:**
- Both total_time and active_time: 10 recipes (e.g., 30/20, 60/40, 90/30)
- Total time only (active=0): 5 recipes (passive cooking)
- Neither (both 0): 5 recipes (no-cook or instant)

**Label Count per Recipe:**
- 0 labels: 2 recipes
- 1 label: 2 recipes
- 2-4 labels: 10 recipes (normal combinations)
- 5-7 labels: 4 recipes (detailed tagging)
- 10+ labels: 2 recipes (stress test)

**Special Combinations:**
- Recipe with vegan + glutenfree
- Recipe with sousvide + main + protein
- Recipe with air fryer + quick
- Very short recipe body (<50 chars)
- Very long recipe body (keep existing long ones)
- Deleted recipe marked as new (edge case)

### Notes Distribution

**Total notes across all recipes:**
- 0 notes: 12 recipes
- 1 note: 4 recipes (mix of flagged and unflagged)
- 2 notes: 3 recipes
- 3+ notes: 1 recipe (for timestamp sorting tests)

**Note characteristics:**
- Keep existing 3 notes
- Add ~10 new notes
- Mix of flagged (1) and unflagged (0) status
- Mostly short test stubs
- One longer note with line breaks
- Varied create_date timestamps for sorting tests

## Implementation Details

### Files to Update
1. `bootstrapping/recipes.csv` - rewrite with 20 recipes
2. `bootstrapping/recipe-label.csv` - rewrite with new label mappings
3. `bootstrapping/notes.csv` - expand from 3 to ~13 notes
4. `bootstrapping/labels.csv` - keep as is (no changes)
5. `bootstrapping/users.csv` - keep as is (no changes)

### Recipe ID Mapping Strategy
- Keep recipe IDs 2 and 10 for preserved recipes
- Reuse other IDs (1, 3-9, 11-13) and add new IDs (14-20)
- Ensure note recipe_id references remain valid

### Label Assignment Strategy
- Use existing label IDs from labels.csv
- Ensure course labels (main=36, side=35, dessert=17, drink=9, breakfast=8, appetizer=12) are properly distributed
- Create realistic combinations (e.g., chicken + main + asian, beef + soup + spicy)
- Include edge cases (no labels, many labels)

### Note Timestamp Strategy
- Use unix timestamps
- Space notes out over several months for realistic sorting
- Multiple notes on same recipe should have different timestamps

## Testing Requirements

After implementation, verify:
1. Bootstrap process completes without errors
2. All 20 recipes load correctly
3. Recipe-label relationships correctly map
4. Notes correctly associate with recipes and sort by create_date
5. Deleted recipes can be queried and undeleted
6. Recipes with 0 labels can be queried
7. Filtering by course labels returns expected counts
8. User authentication works with existing users

## Success Criteria

- [x] 20 recipes total with varied characteristics
- [x] Good distribution across course types (main: 4, side: 3, dessert: 4, drink: 2, breakfast: 5, appetizer: 3)
- [x] At least 2 recipes with no course labels (recipe 19: sauce only, recipe 20: no labels)
- [x] Variety of flag states (new: 5, deleted: 2)
- [x] Realistic time values rounded to 10m
- [x] Label counts ranging from 0 to 12 (recipe 20: 0, recipe 17: 12, recipe 10: 10)
- [x] 13 notes with varied flagged states (3 flagged) and timestamps
- [x] Bootstrap process runs successfully
- [x] All edge cases covered (deleted, no labels, many labels, no notes, many notes)

## Implementation Summary

Successfully updated bootstrap CSV files:
- **recipes.csv**: 20 recipes with varied times, flags, and content
- **recipe-label.csv**: 72 label mappings with distribution from 0-12 labels per recipe
- **notes.csv**: 13 notes across 10 recipes with varied timestamps and flags
- **labels.csv**: No changes (kept intentional test labels like "lamp")
- **users.csv**: No changes (kept existing admin/non-admin users)

Verified data loads correctly through bootstrap process and database queries.
