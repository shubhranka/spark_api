-- migrations/000004_add_test_data.down.sql

-- Remove all test data by truncating the tables
BEGIN;

TRUNCATE users, profiles, interests, user_interests RESTART IDENTITY CASCADE;

COMMIT;