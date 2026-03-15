-- Migration: Add administrator column to user table
-- Date: 2026-03-14
-- Purpose: Add admin authorization support to distinguish admin from regular users

-- Add administrator column if it doesn't exist (idempotent check)
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'user'
  AND COLUMN_NAME = 'administrator';

SET @query = IF(@col_exists = 0,
    'ALTER TABLE user ADD COLUMN administrator BOOLEAN NOT NULL DEFAULT 0',
    'SELECT ''Column already exists'' AS msg');
PREPARE stmt FROM @query;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Set foo (user_id=1) as administrator
UPDATE user SET administrator = 1 WHERE user_id = 1;

-- Verification query (run after migration to confirm)
-- SELECT user_id, username, administrator FROM user;
