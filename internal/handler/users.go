package handler

import (
	"database/sql"
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

// GetMe is a simple handler to test authentication
func GetMe(c *gin.Context) {
	uid := c.MustGet(authorizationPayloadKey).(string)
	c.JSON(http.StatusOK, gin.H{"message": "you are authenticated!", "your_firebase_uid": uid})
}
