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
