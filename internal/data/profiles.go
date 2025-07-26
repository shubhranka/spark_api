package data

import (
	"database/sql"
	"encoding/json"
	"log"
)

type Profile struct {
	Gender            string   `json:"gender"`
	Pronouns          string   `json:"pronouns"`
	SexualOrientation []string `json:"sexual_orientation"`
	GeneralInterests  []string `json:"general_interests"`
	OpeningQuestion   string   `json:"opening_question"`
	Dealbreakers      string   `json:"dealbreakers,omitempty"`
}

type ProfileModel struct {
	DB *sql.DB
}

// CreateOrUpdateProfile handles the full onboarding data for a user.
func (m ProfileModel) CreateOrUpdateProfile(userID string, profileData *Profile) error {
	// We'll use a transaction to ensure all database operations succeed or fail together.
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback the transaction if any step fails

	// 1. Convert sexual_orientation to JSONB for storage
	orientationJSON, err := json.Marshal(profileData.SexualOrientation)
	if err != nil {
		return err
	}

	// 2. Insert or update the main profile data
	// ON CONFLICT (user_id) DO UPDATE is a powerful postgres feature (UPSERT)
	profileQuery := `
		INSERT INTO profiles (user_id, gender, pronouns, sexual_orientation, opening_question, dealbreakers)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			gender = EXCLUDED.gender,
			pronouns = EXCLUDED.pronouns,
			sexual_orientation = EXCLUDED.sexual_orientation,
			opening_question = EXCLUDED.opening_question,
			dealbreakers = EXCLUDED.dealbreakers,
			updated_at = NOW();`
	_, err = tx.Exec(profileQuery, userID, profileData.Gender, profileData.Pronouns, orientationJSON, profileData.OpeningQuestion, profileData.Dealbreakers)
	if err != nil {
		return err
	}

	// 3. Handle general interests
	// First, clear existing interests for this user to handle updates cleanly
	_, err = tx.Exec("DELETE FROM user_interests WHERE user_id = $1", userID)
	if err != nil {
		return err
	}

	// Loop through provided interests
	for _, interestName := range profileData.GeneralInterests {
		var interestID int
		// Check if the interest already exists in the 'interests' table
		err := tx.QueryRow("SELECT id FROM interests WHERE name = $1", interestName).Scan(&interestID)
		if err == sql.ErrNoRows {
			// Interest does not exist, so insert it and get the new ID
			err = tx.QueryRow("INSERT INTO interests (name) VALUES ($1) RETURNING id", interestName).Scan(&interestID)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		// Now, link the user to the interest in the 'user_interests' table
		_, err = tx.Exec("INSERT INTO user_interests (user_id, interest_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", userID, interestID)
		if err != nil {
			return err
		}
	}
	log.Println("Successfully processed profile and interests for user:", userID)
	// If all steps were successful, commit the transaction
	return tx.Commit()
}

// GetProfileByUserID fetches a user's profile information using their internal UUID.
func (m ProfileModel) GetProfileByUserID(userID string) (*Profile, error) {
	// This query will join profiles with an aggregation of user_interests
	query := `
		SELECT
			p.gender,
			p.pronouns,
			p.sexual_orientation,
			p.opening_question,
			p.dealbreakers,
			COALESCE(
				(
					SELECT json_agg(i.name)
					FROM user_interests ui
					JOIN interests i ON ui.interest_id = i.id
					WHERE ui.user_id = p.user_id
				), '[]'::json
			) as general_interests
		FROM profiles p
		WHERE p.user_id = $1;`

	var profile Profile
	var orientationJSON, interestsJSON []byte // Use byte slices to scan JSON data

	err := m.DB.QueryRow(query, userID).Scan(
		&profile.Gender,
		&profile.Pronouns,
		&orientationJSON,
		&profile.OpeningQuestion,
		&profile.Dealbreakers,
		&interestsJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// This is not a server error. It just means the user hasn't completed onboarding.
			return nil, sql.ErrNoRows
		}
		// This is a real database error
		return nil, err
	}

	// Unmarshal the JSON byte slices back into string slices
	if err := json.Unmarshal(orientationJSON, &profile.SexualOrientation); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(interestsJSON, &profile.GeneralInterests); err != nil {
		return nil, err
	}

	return &profile, nil
}
