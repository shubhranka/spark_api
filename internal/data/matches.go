package data

import (
	"database/sql"
	"log"
)

// MatchProfile represents the anonymous data we show for a potential match.
type MatchProfile struct {
	UserID          string `json:"user_id"` // This is the other user's ID
	DisplayName     string `json:"display_name"`
	MatchReason     string `json:"match_reason"`
	OpeningQuestion string `json:"opening_question"`
	// We'll add hasAudioIntro later when we do media uploads.
}

type MatchModel struct {
	DB *sql.DB
}

// GetPotentialMatches finds suitable matches for a given user ID.
func (m MatchModel) GetPotentialMatches(currentUserID string) ([]MatchProfile, error) {
	// This query is the heart of our matching engine.
	// It's complex, so let's break it down:
	// 1. We select from `users` aliased as `u2` (the potential match).
	// 2. We JOIN their profile `p2`.
	// 3. We use subqueries `(SELECT ...)` to get the current user's (`u1`) profile `p1`.
	// 4. The WHERE clause enforces all our matching rules.
	query := `
		SELECT
			u2.id,
			p2.gender,
			p2.opening_question,
			-- This subquery finds ONE shared interest to use as the "match reason"
			(
				SELECT i.name
				FROM user_interests ui1
				JOIN user_interests ui2 ON ui1.interest_id = ui2.interest_id
				JOIN interests i ON ui1.interest_id = i.id
				WHERE ui1.user_id = u1.id AND ui2.user_id = u2.id
				LIMIT 1
			) AS shared_interest
		FROM
			users u1
		JOIN
			profiles p1 ON u1.id = p1.user_id
		JOIN
			users u2 ON u1.id != u2.id -- Rule 1: Not the same user
		JOIN
			profiles p2 ON u2.id = p2.user_id -- Ensure potential match has a profile
		WHERE
			u1.id = $1
			-- Rule 2: The other user's gender is one the current user is interested in.
			-- The '?' operator checks if a string exists in a JSON array.
			AND p1.sexual_orientation ? p2.gender
			-- Rule 3: The current user's gender is one the other user is interested in.
			AND p2.sexual_orientation ? p1.gender
			-- Rule 4: They share at least one interest.
			-- EXISTS is more efficient than a JOIN for just checking existence.
			AND EXISTS (
				SELECT 1
				FROM user_interests ui1
				JOIN user_interests ui2 ON ui1.interest_id = ui2.interest_id
				WHERE ui1.user_id = u1.id AND ui2.user_id = u2.id
			);
	`

	rows, err := m.DB.Query(query, currentUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []MatchProfile
	for rows.Next() {
		var match MatchProfile
		var sharedInterest sql.NullString // Use sql.NullString for safety

		if err := rows.Scan(&match.UserID, &match.DisplayName, &match.OpeningQuestion, &sharedInterest); err != nil {
			log.Printf("Error scanning match row: %v", err)
			continue // Skip problematic rows
		}

		if sharedInterest.Valid {
			match.MatchReason = "Shared interest in " + sharedInterest.String
		} else {
			match.MatchReason = "You have compatible interests"
		}

		matches = append(matches, match)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}
