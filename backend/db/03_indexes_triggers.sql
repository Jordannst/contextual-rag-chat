-- Add full-text search capability and indexes for hybrid search

-- Step 1: Add text_search column with tsvector type for full-text search
ALTER TABLE documents 
ADD COLUMN IF NOT EXISTS text_search tsvector;

-- Step 2: Create GIN index on text_search column for fast text search
-- GIN (Generalized Inverted Index) is optimized for full-text search operations
CREATE INDEX IF NOT EXISTS idx_text_search 
ON documents 
USING GIN(text_search);

-- Step 3: Backfill existing data (if any)
-- Update all existing rows to populate text_search column from content
UPDATE documents 
SET text_search = to_tsvector('english', COALESCE(content, ''))
WHERE text_search IS NULL;

-- Step 4: Create function to automatically update text_search
CREATE OR REPLACE FUNCTION update_text_search()
RETURNS TRIGGER AS $$
BEGIN
    NEW.text_search := to_tsvector('english', COALESCE(NEW.content, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 5: Create trigger to auto-update text_search on insert/update
DROP TRIGGER IF EXISTS trigger_update_text_search ON documents;
CREATE TRIGGER trigger_update_text_search
BEFORE INSERT OR UPDATE OF content ON documents
FOR EACH ROW
EXECUTE FUNCTION update_text_search();

