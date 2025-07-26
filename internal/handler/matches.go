package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shubhranka/spark_api/internal/data"
)

func GetMatches(c *gin.Context) {
	// Get the authenticated user's Firebase UID from the context
	firebaseUID := c.MustGet(authorizationPayloadKey).(string)

	// Get dependencies
	userModel := c.MustGet("userModel").(data.UserModel)
	matchModel := c.MustGet("matchModel").(data.MatchModel)

	// Find our internal user ID from the Firebase UID
	user, err := userModel.GetByFirebaseUID(firebaseUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "authenticated user not found in local database"})
		return
	}

	// Call the data layer to find potential matches
	matches, err := matchModel.GetPotentialMatches(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve matches"})
		return
	}

	// In a real app, you might get a "no matches" screen, but an empty array is RESTfully correct.
	if matches == nil {
		matches = []data.MatchProfile{}
	}

	c.JSON(http.StatusOK, matches)
}
