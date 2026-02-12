-- Migration: Add phone columns to users table
-- This adds support for phone number storage structured for payment gateway integration
-- Columns are nullable to support existing users (migration strategy)

-- Add phone_country_code with default for Brazil
ALTER TABLE users ADD COLUMN phone_country_code TEXT DEFAULT '55';

-- Add phone_area_code (DDD in Brazil)
ALTER TABLE users ADD COLUMN phone_area_code TEXT;

-- Add phone_number (8 or 9 digits)
ALTER TABLE users ADD COLUMN phone_number TEXT;

-- Create index for queries by area code (optional but recommended)
CREATE INDEX IF NOT EXISTS idx_users_phone_area ON users(phone_area_code);
