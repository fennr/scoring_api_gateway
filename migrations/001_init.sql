-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create verifications table
CREATE TABLE IF NOT EXISTS verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inn VARCHAR(12) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'IN_PROCESS',
    author_email VARCHAR(255) NOT NULL,
    requested_data_types TEXT[] NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create verification_data table
CREATE TABLE IF NOT EXISTS verification_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    verification_id UUID NOT NULL REFERENCES verifications(id) ON DELETE CASCADE,
    data_type VARCHAR(50) NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_verifications_inn ON verifications(inn);
CREATE INDEX IF NOT EXISTS idx_verifications_status ON verifications(status);
CREATE INDEX IF NOT EXISTS idx_verification_data_verification_id ON verification_data(verification_id);
CREATE INDEX IF NOT EXISTS idx_verification_data_data_type ON verification_data(data_type); 