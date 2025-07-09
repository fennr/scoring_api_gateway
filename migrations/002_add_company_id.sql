-- Add company_id column to verifications table
ALTER TABLE verifications ADD COLUMN IF NOT EXISTS company_id VARCHAR(255);

-- Create index for company_id
CREATE INDEX IF NOT EXISTS idx_verifications_company_id ON verifications(company_id); 