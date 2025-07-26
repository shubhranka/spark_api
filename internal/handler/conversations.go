package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shubhranka/spark_api/internal/data"
)

type startConversationRequest struct {
	RecipientID string `json:"recipient_id" binding:"required"`
	Content     string `json:"content" binding:"required"`
}

type sendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

func StartConversation(c *gin.Context) {
	var req startConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

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

	// Call the data layer to start the conversation
	conv, err := convModel.Start(currentUser.ID, req.RecipientID, req.Content)
	if err != nil {
		// A more robust error handling would check for specific error types
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not start conversation: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, conv)
}

func SendMessage(c *gin.Context) {
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	conversationID := c.Param("id")
	firebaseUID := c.MustGet(authorizationPayloadKey).(string)

	userModel := c.MustGet("userModel").(data.UserModel)
	convModel := c.MustGet("conversationModel").(data.ConversationModel)

	currentUser, err := userModel.GetByFirebaseUID(firebaseUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "authenticated user not found"})
		return
	}

	// This is the model method we will create next
	msg, err := convModel.AddMessage(conversationID, currentUser.ID, req.Content)
	if err != nil {
		if err.Error() == "cannot send another message until the recipient replies" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not send message: " + err.Error()})
		return
	}

	msgBytes, err := json.Marshal(msg)
	if err == nil {
		WSHub.Broadcast(conversationID, msgBytes)
	}
	c.JSON(http.StatusCreated, msg)
}

func GetConversationDetails(c *gin.Context) {
	conversationID := c.Param("id")
	firebaseUID := c.MustGet(authorizationPayloadKey).(string)

	userModel := c.MustGet("userModel").(data.UserModel)
	convModel := c.MustGet("conversationModel").(data.ConversationModel)

	currentUser, err := userModel.GetByFirebaseUID(firebaseUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "authenticated user not found"})
		return
	}

	details, err := convModel.GetByID(conversationID, currentUser.ID)
	if err != nil {
		// This will catch the "not found or not a participant" error
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, details)
}
