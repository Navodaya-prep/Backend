package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"navodaya-api/handlers"
	"navodaya-api/middleware"
)

func Setup(r *gin.Engine) {
	api := r.Group("/api")

	// Health check
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Navodaya Prime Go API is running 🚀"})
	})

	// Auth routes (public)
	auth := api.Group("/auth")
	{
		auth.POST("/send-otp", handlers.SendOTP)
		auth.POST("/verify-otp", handlers.VerifyOTP)
		auth.POST("/signup", middleware.RequireTempAuth(), handlers.Signup)
	}

	// Admin routes — only X-Admin-Key required, no JWT
	admin := api.Group("/admin")
	admin.Use(middleware.RequireAdmin())
	{
		admin.GET("/mocktests", handlers.ListAdminMockTests)
		admin.POST("/mocktests", handlers.CreateMockTest)
		admin.GET("/mocktests/:id/questions", handlers.ListAdminMockTestQuestions)
		admin.POST("/mocktests/:id/questions", handlers.AddQuestionToMockTest)
	}

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.RequireAuth())
	{
		// Courses
		protected.GET("/courses", handlers.ListCourses)
		protected.GET("/courses/:id", handlers.GetCourse)
		protected.GET("/courses/:id/chapters", handlers.GetCourseChapters)

		// Practice
		protected.GET("/practice/questions/:chapterId", handlers.GetPracticeQuestions)
		protected.POST("/practice/submit", handlers.SubmitPractice)

		// Mock Tests
		protected.GET("/mocktests", handlers.ListMockTests)
		protected.GET("/mocktests/attempts", handlers.GetUserAttempts)
		protected.GET("/mocktests/:id", handlers.GetMockTest)
		protected.POST("/mocktests/:id/submit", handlers.SubmitMockTest)

		// Leaderboard
		protected.GET("/leaderboard", handlers.GetLeaderboard)

		// Profile
		protected.GET("/profile/me", handlers.GetProfile)
		protected.PUT("/profile/update", handlers.UpdateProfile)
	}
}
