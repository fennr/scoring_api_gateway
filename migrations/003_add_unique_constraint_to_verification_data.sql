-- Add a unique constraint on verification_id and data_type in the verification_data table.
-- This is necessary for the INSERT ... ON CONFLICT query to work correctly,
-- ensuring that each data type for a given verification is stored only once.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'verification_data_unique_verification_id_data_type'
    ) THEN
        ALTER TABLE verification_data
        ADD CONSTRAINT verification_data_unique_verification_id_data_type UNIQUE (verification_id, data_type);
    END IF;
END;
$$; 