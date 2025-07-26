package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/api/option"

	"github.com/shubhranka/spark_api/internal/data"    // <-- CHECK YOUR PATH
	"github.com/shubhranka/spark_api/internal/handler" // <-- CHECK YOUR PATH
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Info: .env file not found, relying on environment variables.")
	}

	// Initialize Firebase Admin SDK
	authClient, err := initializeFirebase()
	if err != nil {
		log.Fatalf("Firebase initialization failed: %v", err)
	}
	fmt.Println("Successfully connected to Firebase!")

	// Connect to the database
	db, err := connectDB()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()
	fmt.Println("Successfully connected to the database!")

	// Initialize dependencies
	userModel := data.UserModel{DB: db}
	profileModel := data.ProfileModel{DB: db}
	matchModel := data.MatchModel{DB: db}
	conversationModel := data.ConversationModel{DB: db}

	// Setup Gin router
	router := gin.Default()

	// Middleware to inject dependencies into the context
	router.Use(func(c *gin.Context) {
		c.Set("userModel", userModel)
		c.Set("profileModel", profileModel)
		c.Set("matchModel", matchModel)
		c.Set("conversationModel", conversationModel)
		c.Set("authClient", authClient)
		c.Set("db", db)
		c.Next()
	})

	// Setup routes
	v1 := router.Group("/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		wsRoutes := v1.Group("/ws")
		wsRoutes.Use(func(c *gin.Context) { // A simpler middleware for WS
			c.Set("authClient", authClient)
			c.Set("userModel", userModel)
			c.Set("conversationModel", conversationModel)
			c.Next()
		})
		{
			wsRoutes.GET("/chat/:id", handler.HandleWebSocketConnection)
		}

		// This group handles the initial user sync
		authSyncRoutes := v1.Group("/auth")
		authSyncRoutes.Use(handler.AuthMiddleware(authClient))
		{
			authSyncRoutes.POST("/sync", handler.SyncUser)
		}

		// This group handles all other authenticated actions
		apiRoutes := v1.Group("/")
		apiRoutes.Use(handler.AuthMiddleware(authClient))
		{
			apiRoutes.GET("/me", handler.GetMe)
			apiRoutes.POST("/onboarding", handler.CompleteOnboarding)

			// The new matches route
			apiRoutes.GET("/matches", handler.GetMatches)
			apiRoutes.GET("/users/:id", handler.GetUserProfile)

			convRoutes := apiRoutes.Group("/conversations")
			{
				convRoutes.POST("/start", handler.StartConversation)
				convRoutes.GET("", handler.GetConversations)
				convRoutes.POST("/:id/messages", handler.SendMessage)
				convRoutes.GET("/:id", handler.GetConversationDetails)
				// We will add the other conversation endpoints here in the next steps
			}
		}
	}

	// Start the server
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// connectDB helper function
func connectDB() (*sql.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// initializeFirebase helper function
func initializeFirebase() (*auth.Client, error) {
	keyDataString := os.Getenv("KEY_JSON")
	if keyDataString == "" {
		return nil, fmt.Errorf("KEY_JSON environment variable is not set")
	}
	var keyData map[string]interface{}
	err := json.Unmarshal([]byte(keyDataString), &keyData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling key data: %w", err)
	}
	keyData["private_key"] = strings.ReplaceAll(keyData["private_key"].(string), "\\n", "\n")
	parsedKeyDataString, err := json.Marshal(keyData)
	if err != nil {
		return nil, fmt.Errorf("error marshalling key data: %w", err)
	}
	opt := option.WithCredentialsJSON(parsedKeyDataString)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing app: %w", err)
	}
	client, err := app.Auth(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting Auth client: %w", err)
	}
	return client, nil
}
