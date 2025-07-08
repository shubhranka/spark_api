package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "firebase_uid"
)

// AuthMiddleware creates a gin middleware for Firebase authorization
func AuthMiddleware(authClient *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeader := c.GetHeader(authorizationHeaderKey)

		if len(authorizationHeader) == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header is not provided"})
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unsupported authorization type"})
			return
		}

		idToken := fields[1]
		token, err := authClient.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			fmt.Println("Error verifying ID token:", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid id token"})
			return
		}

		// Set the Firebase UID in the context for subsequent handlers to use
		c.Set(authorizationPayloadKey, token.UID)
		c.Next()
	}
}
