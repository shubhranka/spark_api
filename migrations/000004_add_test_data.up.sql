-- migrations/000004_add_test_data.up.sql

-- Use a transaction to ensure all or nothing
BEGIN;

-- Clear all relevant tables to make this script re-runnable
TRUNCATE users, profiles, interests, user_interests RESTART IDENTITY CASCADE;

-- Declare variables to hold the new user IDs
DO $$
DECLARE
    alice_id UUID;
    bob_id UUID;
    carol_id UUID;
    dave_id UUID;
    eve_id UUID;
    frank_id UUID;
    hiking_id INT;
    coding_id INT;
    photo_id INT;
    cooking_id INT;
    movies_id INT;
    board_games_id INT;
    live_music_id INT;
    video_games_id INT;
    art_id INT;
    yoga_id INT;
    sailing_id INT;
    gardening_id INT;
BEGIN

    -- Insert users and capture their generated UUIDs
    INSERT INTO users (firebase_uid, email, display_name) VALUES ('test-uid-alice', 'alice@test.com', 'Alice') RETURNING id INTO alice_id;
    INSERT INTO users (firebase_uid, email, display_name) VALUES ('test-uid-bob', 'bob@test.com', 'Bob') RETURNING id INTO bob_id;
    INSERT INTO users (firebase_uid, email, display_name) VALUES ('test-uid-carol', 'carol@test.com', 'Carol') RETURNING id INTO carol_id;
    INSERT INTO users (firebase_uid, email, display_name) VALUES ('test-uid-dave', 'dave@test.com', 'Dave') RETURNING id INTO dave_id;
    INSERT INTO users (firebase_uid, email, display_name) VALUES ('test-uid-eve', 'eve@test.com', 'Eve') RETURNING id INTO eve_id;
    INSERT INTO users (firebase_uid, email, display_name) VALUES ('test-uid-frank', 'frank@test.com', 'Frank') RETURNING id INTO frank_id;

    -- Insert profiles for each user
    -- Alice: Woman into women
    INSERT INTO profiles (user_id, gender, pronouns, sexual_orientation, opening_question) VALUES (alice_id, 'Woman', 'she/her', '["Woman"]'::jsonb, 'Best trail ever?');
    -- Bob: Man into women
    INSERT INTO profiles (user_id, gender, pronouns, sexual_orientation, opening_question) VALUES (bob_id, 'Man', 'he/him', '["Woman"]'::jsonb, 'Dish you could eat forever?');
    -- Carol: Woman into men and women
    INSERT INTO profiles (user_id, gender, pronouns, sexual_orientation, opening_question) VALUES (carol_id, 'Woman', 'she/her', '["Man", "Woman"]'::jsonb, 'Favorite concert?');
    -- Dave: Man into men
    INSERT INTO profiles (user_id, gender, pronouns, sexual_orientation, opening_question) VALUES (dave_id, 'Man', 'he/him', '["Man"]'::jsonb, 'Language to love/hate?');
    -- Eve: Woman into women and non-binary people
    INSERT INTO profiles (user_id, gender, pronouns, sexual_orientation, opening_question) VALUES (eve_id, 'Woman', 'she/they', '["Woman", "Non-binary"]'::jsonb, 'Favorite yoga pose?');
    -- Frank: Man into men
    INSERT INTO profiles (user_id, gender, pronouns, sexual_orientation, opening_question) VALUES (frank_id, 'Man', 'he/him', '["Man"]'::jsonb, 'What are you growing?');

    -- Insert all unique interests and get their IDs
    INSERT INTO interests (name) VALUES ('Hiking') RETURNING id INTO hiking_id;
    INSERT INTO interests (name) VALUES ('Coding') RETURNING id INTO coding_id;
    INSERT INTO interests (name) VALUES ('Photography') RETURNING id INTO photo_id;
    INSERT INTO interests (name) VALUES ('Cooking') RETURNING id INTO cooking_id;
    INSERT INTO interests (name) VALUES ('Movies') RETURNING id INTO movies_id;
    INSERT INTO interests (name) VALUES ('Board Games') RETURNING id INTO board_games_id;
    INSERT INTO interests (name) VALUES ('Live Music') RETURNING id INTO live_music_id;
    INSERT INTO interests (name) VALUES ('Video Games') RETURNING id INTO video_games_id;
    INSERT INTO interests (name) VALUES ('Art') RETURNING id INTO art_id;
    INSERT INTO interests (name) VALUES ('Yoga') RETURNING id INTO yoga_id;
    INSERT INTO interests (name) VALUES ('Sailing') RETURNING id INTO sailing_id;
    INSERT INTO interests (name) VALUES ('Gardening') RETURNING id INTO gardening_id;

    -- Link users to their interests
    INSERT INTO user_interests (user_id, interest_id) VALUES
        -- Alice
        (alice_id, hiking_id), (alice_id, coding_id), (alice_id, photo_id),
        -- Bob
        (bob_id, cooking_id), (bob_id, movies_id), (bob_id, board_games_id),
        -- Carol
        (carol_id, hiking_id), (carol_id, cooking_id), (carol_id, live_music_id),
        -- Dave
        (dave_id, coding_id), (dave_id, movies_id), (dave_id, video_games_id),
        -- Eve
        (eve_id, hiking_id), (eve_id, art_id), (eve_id, yoga_id),
        -- Frank
        (frank_id, sailing_id), (frank_id, gardening_id);

END $$;

COMMIT;