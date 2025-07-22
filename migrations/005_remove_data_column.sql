-- Migration 005: Remove obsolete data column from verification_data
-- After cache system implementation, data is stored in verification_data_cache
-- and verification_data only needs data_hash column

-- Safety check: ensure all records have data_hash before removing data column
DO $$
DECLARE
    records_without_hash INTEGER;
BEGIN
    SELECT COUNT(*) INTO records_without_hash 
    FROM verification_data 
    WHERE data_hash IS NULL AND data IS NOT NULL;
    
    IF records_without_hash > 0 THEN
        RAISE NOTICE 'Found % records without data_hash. Running migration function first...', records_without_hash;
        PERFORM migrate_existing_verification_data();
        
        -- Recheck after migration
        SELECT COUNT(*) INTO records_without_hash 
        FROM verification_data 
        WHERE data_hash IS NULL AND data IS NOT NULL;
        
        IF records_without_hash > 0 THEN
            RAISE EXCEPTION 'Cannot remove data column: % records still without data_hash', records_without_hash;
        END IF;
    END IF;
    
    RAISE NOTICE 'All records have data_hash. Safe to remove data column.';
END $$;

-- Remove the obsolete data column
ALTER TABLE verification_data DROP COLUMN IF EXISTS data;

-- Add comment for clarity
COMMENT ON TABLE verification_data IS 'Verification data references - actual data stored in verification_data_cache';
COMMENT ON COLUMN verification_data.data_hash IS 'SHA-256 hash referencing data in verification_data_cache table'; 