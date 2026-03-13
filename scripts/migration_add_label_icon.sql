-- Migration: Add icon column to label table
-- Date: 2026-03-11
-- Purpose: Add emoji/character icon support to labels

-- CRITICAL: Set connection charset to utf8mb4 for proper emoji handling
-- Without this, emojis will be double-encoded and appear as garbled text
SET NAMES utf8mb4;

-- Ensure table uses utf8mb4 charset (required for 4-byte emojis)
-- MySQL's utf8/utf8mb3 only supports 3-byte chars and will corrupt emojis
ALTER TABLE label CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Add icon column if it doesn't exist (idempotent check)
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'label'
  AND COLUMN_NAME = 'icon';

SET @query = IF(@col_exists = 0,
    'ALTER TABLE label ADD COLUMN icon VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''''',
    'SELECT ''Column already exists'' AS msg');
PREPARE stmt FROM @query;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Populate icons for existing labels based on production_labels.csv mappings
UPDATE label SET icon = '🐄' WHERE label_id = 1;   -- Beef
UPDATE label SET icon = '🐓' WHERE label_id = 2;   -- Chicken
UPDATE label SET icon = '🐖' WHERE label_id = 3;   -- Pork
UPDATE label SET icon = '🐟' WHERE label_id = 4;   -- Fish
UPDATE label SET icon = '🐑' WHERE label_id = 5;   -- Lamb
UPDATE label SET icon = '🥦' WHERE label_id = 6;   -- Vegetarian
UPDATE label SET icon = 'Ⓥ' WHERE label_id = 7;   -- Vegan
UPDATE label SET icon = '🍳' WHERE label_id = 8;   -- Breakfast
UPDATE label SET icon = '🥤' WHERE label_id = 9;   -- Drink
UPDATE label SET icon = '🍜' WHERE label_id = 10;  -- SoupStew
UPDATE label SET icon = '🥬' WHERE label_id = 11;  -- Salad
UPDATE label SET icon = '🥟' WHERE label_id = 12;  -- Appetizer
UPDATE label SET icon = '🥪' WHERE label_id = 13;  -- Sandwich
UPDATE label SET icon = '🇲🇽' WHERE label_id = 14;  -- Mexican
UPDATE label SET icon = '🥢' WHERE label_id = 15;  -- Asian
-- MiddleEast (16) - no icon
UPDATE label SET icon = '🍦' WHERE label_id = 17;  -- Dessert
UPDATE label SET icon = '🍞' WHERE label_id = 18;  -- Bread
UPDATE label SET icon = '🥕' WHERE label_id = 19;  -- Vegetable
UPDATE label SET icon = '🍪' WHERE label_id = 20;  -- Cookie
UPDATE label SET icon = '🎂' WHERE label_id = 21;  -- Cake
UPDATE label SET icon = '🍬' WHERE label_id = 22;  -- Candy
-- Cheesecake (23) - no icon
-- CreamCustard (24) - no icon
UPDATE label SET icon = '🍏' WHERE label_id = 25;  -- Fruit
UPDATE label SET icon = 'Ⓖ' WHERE label_id = 26;  -- GlutenFree
UPDATE label SET icon = '🍝' WHERE label_id = 27;  -- Pasta
UPDATE label SET icon = '🇬🇷' WHERE label_id = 28;  -- Greek
UPDATE label SET icon = '🌶️' WHERE label_id = 29;  -- Spicy
UPDATE label SET icon = '⚡' WHERE label_id = 30;  -- Quick
UPDATE label SET icon = '🦐' WHERE label_id = 31;  -- Shrimp
-- Sauce (32) - no icon
UPDATE label SET icon = '🍚' WHERE label_id = 33;  -- Rice
-- StarchSide (34) - no icon
-- Side (35) - no icon
UPDATE label SET icon = '🍽️' WHERE label_id = 36;  -- Main
UPDATE label SET icon = '🦃' WHERE label_id = 37;  -- Turkey
UPDATE label SET icon = '⏰' WHERE label_id = 38;  -- Batch
-- lamp (39) - no icon (typo/duplicate)
UPDATE label SET icon = '🌡️' WHERE label_id = 40;  -- sousvide
-- air fryer (41) - no icon
UPDATE label SET icon = '🥣' WHERE label_id = 42;  -- soup
-- summer (43) - no icon
-- sauces (44) - no icon (duplicate)
-- tofu (45) - no icon
-- eatsy (46) - no icon

-- Verification query (run after migration to confirm)
-- SELECT label_id, label, icon FROM label WHERE icon != '' ORDER BY label_id;
