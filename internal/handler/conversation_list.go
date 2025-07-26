package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shubhranka/spark_api/internal/data"
)

func GetConversations(c *gin.Context) {
	// Get dependencies
	firebaseUID := c.MustGet(authorizationPayloadKey).(string)
	userModel := c.MustGet("userModel").(data.UserModel)
	convModel := c.MustGet("conversationModel").(data.ConversationModel)

	// Get our internal user ID from the Firebase UID
	currentUser, err := userModel.GetByFirebaseUID(firebaseUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "authenticated user not found"})
		return
	}

	conversations, err := convModel.GetAllForUser(currentUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve conversations"})
		return
	}

	if conversations == nil {
		conversations = []data.ConversationPreview{}
	}

	c.JSON(http.StatusOK, conversations)
}
