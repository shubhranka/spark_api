DROP TRIGGER IF EXISTS on_new_message ON messages;
DROP FUNCTION IF EXISTS update_conversation_timestamp();
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
DROP TYPE IF EXISTS conversation_status;