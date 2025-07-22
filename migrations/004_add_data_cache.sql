-- Migration 004: Add data caching system with hash-based deduplication
-- This migration is idempotent and safe for repeated runs

-- Enable pgcrypto extension for hash functions
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Create verification_data_cache table to store unique JSON data with hashes
CREATE TABLE IF NOT EXISTS verification_data_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data_hash VARCHAR(64) NOT NULL UNIQUE,
    data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add data_hash column to verification_data table if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'verification_data' AND column_name = 'data_hash'
    ) THEN
        ALTER TABLE verification_data ADD COLUMN data_hash VARCHAR(64);
    END IF;
END $$;

-- Create indexes if they don't exist
CREATE INDEX IF NOT EXISTS idx_verification_data_cache_hash ON verification_data_cache(data_hash);
CREATE INDEX IF NOT EXISTS idx_verification_data_hash ON verification_data(data_hash);

-- Create or replace migration function for existing data
CREATE OR REPLACE FUNCTION migrate_existing_verification_data() 
RETURNS void AS $$
DECLARE
    rec RECORD;
    computed_hash VARCHAR(64);
    processed_count INTEGER := 0;
BEGIN
    RAISE NOTICE 'Starting migration of existing verification data...';
    
    FOR rec IN SELECT id, data FROM verification_data WHERE data_hash IS NULL AND data IS NOT NULL LOOP
        computed_hash := encode(digest(rec.data::text, 'sha256'), 'hex');
        
        INSERT INTO verification_data_cache (data_hash, data)
        VALUES (computed_hash, rec.data)
        ON CONFLICT (data_hash) DO NOTHING;
        
        UPDATE verification_data 
        SET data_hash = computed_hash 
        WHERE id = rec.id;
        
        processed_count := processed_count + 1;
        
        IF processed_count % 100 = 0 THEN
            RAISE NOTICE 'Processed % records...', processed_count;
        END IF;
    END LOOP;
    
    RAISE NOTICE 'Migration completed. Processed % records total.', processed_count;
END;
$$ LANGUAGE plpgsql;

-- Automatically run migration for existing data
SELECT migrate_existing_verification_data(); 