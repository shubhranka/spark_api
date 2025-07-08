package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shubhranka/spark_api/internal/data" // <-- IMPORTANT: Change this import path
)

// OnboardingRequest defines the structure for the onboarding data payload.
type OnboardingRequest struct {
	Gender            string   `json:"gender" binding:"required"`
	Pronouns          string   `json:"pronouns"` // Optional
	SexualOrientation []string `json:"sexual_orientation" binding:"required"`
	GeneralInterests  []string `json:"general_interests" binding:"required"`
	OpeningQuestion   string   `json:"opening_question" binding:"required"`
	Dealbreakers      string   `json:"dealbreakers"` // Optional
}

// CompleteOnboarding is the handler function for our new endpoint.
func CompleteOnboarding(c *gin.Context) {
	var req OnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload: " + err.Error()})
		return
	}

	// Get the authenticated user's ID from the context (set by the AuthMiddleware)
	// We need to fetch our internal user ID, not the Firebase UID.
	firebaseUID := c.MustGet(authorizationPayloadKey).(string)
	userModel := c.MustGet("userModel").(data.UserModel)

	fmt.Println("Authenticated user's Firebase UID:", firebaseUID, userModel)

	user, err := userModel.GetByFirebaseUID(firebaseUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "authenticated user not found in local database"})
		return
	}

	// Get the ProfileModel dependency
	profileModel := c.MustGet("profileModel").(data.ProfileModel)

	// Prepare the profile data from the request
	profileData := &data.Profile{
		Gender:            req.Gender,
		Pronouns:          req.Pronouns,
		SexualOrientation: req.SexualOrientation,
		GeneralInterests:  req.GeneralInterests,
		OpeningQuestion:   req.OpeningQuestion,
		Dealbreakers:      req.Dealbreakers,
	}

	// Call the data layer to create or update the profile
	if err := profileModel.CreateOrUpdateProfile(user.ID, profileData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save onboarding data: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "onboarding completed successfully"})
}
