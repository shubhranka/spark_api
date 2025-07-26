package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shubhranka/spark_api/internal/data" // <-- CHECK YOUR PATH
)

// GetUserProfile fetches the public profile for a specific user ID.
func GetUserProfile(c *gin.Context) {
	// Get the user ID from the URL parameter
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID is required"})
		return
	}

	// Get dependencies from context
	userModel := c.MustGet("userModel").(data.UserModel) // We need this for the display name
	profileModel := c.MustGet("profileModel").(data.ProfileModel)

	// Define the structure for our public JSON response.
	// It's very similar to the GetMe response, but we might want to customize it later.
	type PublicUserProfile struct {
		ID                string        `json:"id"`
		DisplayName       string        `json:"display_name"`
		OnboardingProfile *data.Profile `json:"onboarding_profile"`
	}

	// 1. Fetch basic user info (like display_name) by their internal UUID
	user, err := userModel.GetByID(userID) // We need to add this method to our model!
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error fetching user"})
		return
	}

	// 2. Fetch the detailed profile info
	profile, err := profileModel.GetProfileByUserID(user.ID)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user profile"})
		return
	}

	// 3. Assemble the response
	response := PublicUserProfile{
		ID:                user.ID,
		DisplayName:       user.DisplayName,
		OnboardingProfile: profile,
	}

	c.JSON(http.StatusOK, response)
}
