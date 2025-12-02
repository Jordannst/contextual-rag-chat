-- =====================================================
-- SIMPLE VERSION: For manual execution in pgAdmin
-- =====================================================
-- Copy and paste these queries one by one in pgAdmin
-- =====================================================

-- Query 1: Add text_search column
ALTER TABLE documents 
ADD COLUMN IF NOT EXISTS text_search tsvector;

-- Query 2: Create GIN index for fast text search
CREATE INDEX IF NOT EXISTS idx_text_search 
ON documents 
USING GIN(text_search);

-- Query 3: Backfill existing data (this may take time if you have many documents)
UPDATE documents 
SET text_search = to_tsvector('english', COALESCE(content, ''))
WHERE text_search IS NULL;

-- =====================================================
-- OPTIONAL: Auto-update trigger (recommended)
-- =====================================================
-- This ensures text_search is automatically updated when content changes

-- Create function
CREATE OR REPLACE FUNCTION update_text_search()
RETURNS TRIGGER AS $$
BEGIN
    NEW.text_search := to_tsvector('english', COALESCE(NEW.content, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
DROP TRIGGER IF EXISTS trigger_update_text_search ON documents;
CREATE TRIGGER trigger_update_text_search
BEFORE INSERT OR UPDATE OF content ON documents
FOR EACH ROW
EXECUTE FUNCTION update_text_search();

-- =====================================================
-- VERIFICATION (optional)
-- =====================================================
-- Run this to verify the migration was successful:
-- SELECT 
--     id, 
--     LENGTH(content) as content_length,
--     text_search IS NOT NULL as has_text_search,
--     source_file
-- FROM documents 
-- LIMIT 10;

