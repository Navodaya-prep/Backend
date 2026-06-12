package main

import (
	"fmt"
	"log"
	"os"

	"github.com/navodayasarthi/api/config"
	"github.com/navodayasarthi/api/routes"
	"github.com/navodayasarthi/api/utils"
	"github.com/navodayasarthi/api/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Connect MongoDB
	config.ConnectDB()

	// Start WebSocket hub
	go ws.GlobalHub.Run()

	// Start scheduled push notifications (daily challenge, streak reminders)
	go utils.StartNotificationScheduler()

	// Setup Gin
	r := gin.Default()

	// CORS — allow React Native app and admin portal
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Admin-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	// Register all routes
	routes.Setup(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("\n🚀 NavodayaSarthi Go API running on http://localhost:%s\n", port)
	fmt.Printf("📋 Health check: http://localhost:%s/api/health\n\n", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
