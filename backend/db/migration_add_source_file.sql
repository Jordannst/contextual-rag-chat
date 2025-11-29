-- Migration to add source_file column to existing documents table
-- Run this if you already have the documents table without source_file column

ALTER TABLE documents 
ADD COLUMN IF NOT EXISTS source_file VARCHAR(255);

