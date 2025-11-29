-- Migration to add text_search column for Hybrid Search
-- This migration adds full-text search capability using PostgreSQL tsvector
-- Run this migration to enable hybrid search (vector + full-text search)

-- Step 1: Add text_search column with tsvector type
-- This column will store the full-text search index of the content
ALTER TABLE documents 
ADD COLUMN IF NOT EXISTS text_search tsvector;

-- Step 2: Create GIN index on text_search column for fast text search
-- GIN (Generalized Inverted Index) is optimized for full-text search operations
-- This index will significantly speed up text search queries
CREATE INDEX IF NOT EXISTS idx_text_search 
ON documents 
USING GIN(text_search);

-- Step 3: Backfill existing data
-- Update all existing rows to populate text_search column from content
-- Using 'english' language configuration (can be changed to 'indonesian' if needed)
-- This will process all existing documents and create their text search vectors
UPDATE documents 
SET text_search = to_tsvector('english', COALESCE(content, ''))
WHERE text_search IS NULL;

-- Optional: Create a trigger to automatically update text_search when content changes
-- This ensures that new inserts and updates will automatically populate text_search
CREATE OR REPLACE FUNCTION update_text_search()
RETURNS TRIGGER AS $$
BEGIN
    NEW.text_search := to_tsvector('english', COALESCE(NEW.content, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop trigger if exists to avoid conflicts
DROP TRIGGER IF EXISTS trigger_update_text_search ON documents;

-- Create trigger that fires before insert or update
CREATE TRIGGER trigger_update_text_search
BEFORE INSERT OR UPDATE OF content ON documents
FOR EACH ROW
EXECUTE FUNCTION update_text_search();

-- Verification query (optional - run this to check the migration)
-- SELECT 
--     id, 
--     LENGTH(content) as content_length,
--     LENGTH(text_search::text) as text_search_length,
--     source_file
-- FROM documents 
-- LIMIT 10;

