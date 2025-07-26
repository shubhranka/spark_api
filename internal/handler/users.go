package handler

import (
	"database/sql"
	"log"
	"net/http"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/shubhranka/spark_api/internal/data" // Make sure this path is correct
)

// SyncUser checks if a user from a valid token exists in our DB. If not, it creates them.
// It now also returns whether the user's onboarding is complete.
func SyncUser(c *gin.Context) {
	firebaseUID := c.MustGet(authorizationPayloadKey).(string)

	// Get dependencies
	userModel := c.MustGet("userModel").(data.UserModel)
	authClient := c.MustGet("authClient").(*auth.Client) // Assuming you've defined AuthClient elsewhere
	db := c.MustGet("db").(*sql.DB)                      // Get the raw DB connection to check profiles

	// Check if the user already exists in our users table
	user, err := userModel.GetByFirebaseUID(firebaseUID)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error on user check"})
		return
	}

	isNewUser := false
	// If user does not exist, create them
	if err == sql.ErrNoRows {
		isNewUser = true
		firebaseUser, fbErr := authClient.GetUser(c, firebaseUID)
		if fbErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve user from firebase"})
			return
		}

		var displayName sql.NullString
		if firebaseUser.DisplayName != "" {
			displayName.String = firebaseUser.DisplayName
			displayName.Valid = true
		}

		newUser := &data.User{
			FirebaseUID: firebaseUser.UID,
			Email:       firebaseUser.Email,
			DisplayName: displayName.String,
		}

		if insertErr := userModel.Insert(newUser); insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user in local db"})
			return
		}
		// After inserting, we need to get the full user object with the new ID
		user, err = userModel.GetByFirebaseUID(firebaseUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve newly created user"})
			return
		}
	}

	// Now, check if a profile exists for this user (whether they are new or existing)
	var profileExists bool
	profileCheckQuery := `SELECT EXISTS(SELECT 1 FROM profiles WHERE user_id = $1)`
	err = db.QueryRow(profileCheckQuery, user.ID).Scan(&profileExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error on profile check"})
		return
	}

	// The status code depends on whether we created a new user record
	statusCode := http.StatusOK
	if isNewUser {
		statusCode = http.StatusCreated
	}

	c.JSON(statusCode, gin.H{
		"user":                   user,
		"is_onboarding_complete": profileExists,
	})
}

// GetMe fetches the full profile for the authenticated user.
func GetMe(c *gin.Context) {
	// Get dependencies from context
	firebaseUID := c.MustGet(authorizationPayloadKey).(string)
	userModel := c.MustGet("userModel").(data.UserModel)
	profileModel := c.MustGet("profileModel").(data.ProfileModel)

	// Define the structure for our JSON response
	type FullUserProfile struct {
		ID                string        `json:"id"`
		FirebaseUID       string        `json:"firebase_uid"`
		Email             string        `json:"email"`
		DisplayName       string        `json:"display_name"`
		OnboardingProfile *data.Profile `json:"onboarding_profile"` // Use a pointer so it can be null
	}

	// 1. Fetch the basic user info
	user, err := userModel.GetByFirebaseUID(firebaseUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found in local db"})
		return
	}

	// 2. Fetch the detailed profile info
	profile, err := profileModel.GetProfileByUserID(user.ID)
	if err != nil && err != sql.ErrNoRows {
		log.Println("Error fetching profile:", err)
		// This is a real server error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user profile"})
		return
	}
	// If err is sql.ErrNoRows, profile will be nil, which is exactly what we want.

	// 3. Assemble the response
	response := FullUserProfile{
		ID:                user.ID,
		FirebaseUID:       user.FirebaseUID,
		Email:             user.Email,
		DisplayName:       user.DisplayName,
		OnboardingProfile: profile, // This will be null if no profile was found
	}

	c.JSON(http.StatusOK, response)
}
