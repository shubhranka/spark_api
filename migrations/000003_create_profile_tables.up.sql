CREATE TABLE profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    gender TEXT,
    pronouns TEXT,
    sexual_orientation JSONB,
    opening_question TEXT,
    dealbreakers TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add a trigger to automatically update the updated_at timestamp
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_timestamp
BEFORE UPDATE ON profiles
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

-- Table to store unique interests
CREATE TABLE interests (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

-- Join table to link users and interests
CREATE TABLE user_interests (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    interest_id INTEGER NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, interest_id) -- Ensures a user can't have the same interest twice
);

-- Create indexes for faster lookups
CREATE INDEX ON profiles(user_id);
CREATE INDEX ON user_interests(user_id);
CREATE INDEX ON user_interests(interest_id);