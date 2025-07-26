CREATE TYPE conversation_status AS ENUM ('pending', 'active', 'blocked');

CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- This constraint prevents duplicate pending conversations
    UNIQUE (user_a_id, user_b_id),

    status conversation_status NOT NULL DEFAULT 'pending',

    -- Progress tracking fields
    message_count INTEGER NOT NULL DEFAULT 0,
    photos_unlocked BOOLEAN NOT NULL DEFAULT FALSE,
    names_unlocked BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_opening_message BOOLEAN NOT NULL DEFAULT FALSE,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add a trigger to update the conversation's updated_at timestamp on new messages
CREATE OR REPLACE FUNCTION update_conversation_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  UPDATE conversations
  SET updated_at = NOW()
  WHERE id = NEW.conversation_id;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER on_new_message
AFTER INSERT ON messages
FOR EACH ROW
EXECUTE PROCEDURE update_conversation_timestamp();


-- Create indexes for performance
CREATE INDEX ON conversations(user_a_id);
CREATE INDEX ON conversations(user_b_id);
CREATE INDEX ON messages(conversation_id);
CREATE INDEX ON messages(sender_id);