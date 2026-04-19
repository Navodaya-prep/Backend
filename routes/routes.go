package routes

import (
	"net/http"

	"navodaya-api/handlers"
	"navodaya-api/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine) {
	// Serve uploaded images
	r.Static("/uploads", "./uploads")

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

	// Admin routes — X-Admin-Key required
	admin := api.Group("/admin")

	// Admin auth routes (public)
	adminAuth := admin.Group("/auth")
	{
		adminAuth.POST("/login", handlers.AdminLogin)
	}

	// Protected admin routes (JWT required)
	admin.Use(middleware.RequireAdmin())
	{
		// Image upload
		admin.POST("/upload/image", handlers.UploadImage)

		// Admin profile
		admin.GET("/auth/profile", handlers.GetAdminProfile)
		admin.PUT("/auth/profile", handlers.UpdateAdminProfile)
		admin.PUT("/auth/change-password", handlers.ChangeAdminPassword)

		// Admin management (Super Admin only)
		adminManage := admin.Group("/manage")
		adminManage.Use(middleware.RequireSuperAdmin())
		{
			adminManage.GET("/admins", handlers.ListAdmins)
			adminManage.POST("/admins/invite", handlers.InviteAdmin)
			adminManage.DELETE("/admins/:id", handlers.DeleteAdmin)

			// Teacher management
			adminManage.GET("/teachers", handlers.ListTeachers)
			adminManage.POST("/teachers/invite", handlers.InviteTeacher)
			adminManage.PUT("/teachers/:id", handlers.UpdateTeacher)
			adminManage.PUT("/teachers/:id/toggle", handlers.ToggleTeacherStatus)
			adminManage.DELETE("/teachers/:id", handlers.DeleteTeacher)
		}

		// Mock Tests
		admin.GET("/mocktests", handlers.ListAdminMockTests)
		admin.POST("/mocktests", handlers.CreateMockTest)
		admin.PUT("/mocktests/:id", handlers.UpdateMockTest)
		admin.DELETE("/mocktests/:id", handlers.DeleteMockTest)
		admin.GET("/mocktests/:id/questions", handlers.ListAdminMockTestQuestions)
		admin.POST("/mocktests/:id/questions", handlers.AddQuestionToMockTest)
		admin.PUT("/mocktests/:id/questions/reorder", handlers.ReorderMockTestQuestions)
		admin.PUT("/mocktests/:id/questions/:questionId", handlers.UpdateMockTestQuestion)
		admin.DELETE("/mocktests/:id/questions/:questionId", handlers.DeleteMockTestQuestion)

		// Live Classes
		admin.GET("/live/classes", handlers.ListAdminLiveClasses)
		admin.POST("/live/classes", handlers.CreateLiveClass)
		admin.DELETE("/live/classes/:id", handlers.EndLiveClass)
		admin.POST("/live/classes/:id/questions", handlers.PushLiveQuestion)
		admin.DELETE("/live/classes/:id/questions/:qid", handlers.EndLiveQuestion)
		admin.GET("/live/classes/:id/questions/:qid/leaderboard", handlers.GetQuestionLeaderboard)

		// Practice Hub — Subjects
		admin.GET("/practice/subjects", handlers.AdminListSubjects)
		admin.POST("/practice/subjects", handlers.AdminCreateSubject)
		admin.PUT("/practice/subjects/:id", handlers.AdminUpdateSubject)
		admin.DELETE("/practice/subjects/:id", handlers.AdminDeleteSubject)

		// Practice Hub — Chapters (scoped under subject)
		admin.GET("/practice/subjects/:id/chapters", handlers.AdminListChapters)
		admin.POST("/practice/subjects/:id/chapters", handlers.AdminCreateChapter)
		admin.PUT("/practice/chapters/:id", handlers.AdminUpdateChapter)
		admin.DELETE("/practice/chapters/:id", handlers.AdminDeleteChapter)

		// Practice Hub — Questions (scoped under chapter)
		admin.GET("/practice/chapters/:id/questions", handlers.AdminListChapterQuestions)
		admin.POST("/practice/chapters/:id/questions", handlers.AdminCreateQuestion)
		admin.PUT("/practice/questions/:id", handlers.AdminUpdateQuestion)
		admin.DELETE("/practice/questions/:id", handlers.AdminDeleteQuestion)

		// Recorded Classes — Courses
		admin.GET("/courses", handlers.AdminListCourses)
		admin.POST("/courses", handlers.AdminCreateCourse)
		admin.PUT("/courses/:id", handlers.AdminUpdateCourse)
		admin.DELETE("/courses/:id", handlers.AdminDeleteCourse)

		// Recorded Classes — Chapters (scoped under course)
		admin.GET("/courses/:id/chapters", handlers.AdminListCourseChapters)
		admin.POST("/courses/:id/chapters", handlers.AdminCreateCourseChapter)
		// PUT/DELETE /admin/chapters/:id reuses practice hub handlers (shared Chapter model)

		// Recorded Classes — Lessons (scoped under chapter)
		admin.GET("/chapters/:id/lessons", handlers.AdminListLessons)
		admin.POST("/chapters/:id/lessons", handlers.AdminCreateLesson)
		admin.PUT("/lessons/:id", handlers.AdminUpdateLesson)
		admin.DELETE("/lessons/:id", handlers.AdminDeleteLesson)

		// Settings (all admins can view, only super admin can update)
		admin.GET("/settings", handlers.GetSettings)

		// Doubts management
		admin.GET("/doubts", handlers.AdminListDoubts)
		admin.GET("/doubts/:id/answers", handlers.AdminGetDoubtAnswers)
		admin.POST("/doubts/:id/answers", handlers.AdminAnswerDoubt)
		admin.DELETE("/doubts/:id", handlers.AdminDeleteDoubt)

		// Daily Challenge (Super Admin only)
		dailyAdmin := admin.Group("/daily-challenge")
		dailyAdmin.Use(middleware.RequireSuperAdmin())
		{
			dailyAdmin.GET("", handlers.AdminListChallenges)
			dailyAdmin.POST("", handlers.AdminCreateChallenge)
			dailyAdmin.PUT("/:id", handlers.AdminUpdateChallenge)
			dailyAdmin.DELETE("/:id", handlers.AdminDeleteChallenge)
		}
	}

	// Super Admin only settings
	superAdminSettings := api.Group("/admin/settings")
	superAdminSettings.Use(middleware.RequireAdmin(), middleware.RequireSuperAdmin())
	{
		superAdminSettings.PUT("", handlers.UpdateSettings)
	}

	// Protected routes — JWT required
	protected := api.Group("/")
	protected.Use(middleware.RequireAuth())
	protected.Use(middleware.TrackActivity()) // Track user activity for streak
	{
		// Courses
		protected.GET("/courses", handlers.ListCourses)
		protected.GET("/courses/:id", handlers.GetCourse)
		protected.GET("/courses/:id/chapters", handlers.GetCourseChapters)
		protected.GET("/courses/:id/chapters/progress", handlers.GetCourseChaptersWithProgress)
		protected.GET("/chapters/:id/lessons", handlers.GetChapterLessons)
		protected.POST("/lessons/:id/complete", handlers.MarkLessonComplete)

		// Practice (legacy)
		protected.GET("/practice/questions/:chapterId", handlers.GetPracticeQuestions)
		protected.POST("/practice/submit", handlers.SubmitPractice)

		// Practice Hub (student)
		protected.GET("/practice/subjects", handlers.ListSubjects)
		protected.GET("/practice/subjects/:id/chapters", handlers.ListSubjectChapters)
		protected.GET("/practice/chapters/:id/questions", handlers.GetChapterQuestions)
		protected.POST("/practice/chapters/:id/submit", handlers.SubmitChapterPractice)

		// Mock Tests
		protected.GET("/mocktests", handlers.ListMockTests)
		protected.GET("/mocktests/attempts", handlers.GetUserAttempts)
		protected.GET("/mocktests/attempts/:attemptId", handlers.GetAttemptDetails)
		protected.GET("/mocktests/:id", handlers.GetMockTest)
		protected.POST("/mocktests/:id/submit", handlers.SubmitMockTest)

		// Leaderboard
		protected.GET("/leaderboard", handlers.GetLeaderboard)

		// Profile
		protected.GET("/profile/me", handlers.GetProfile)
		protected.PUT("/profile/update", handlers.UpdateProfile)

		// Settings (read-only for students)
		protected.GET("/settings", handlers.GetSettings)

		// Daily Challenge (student)
		protected.GET("/daily-challenge/today", handlers.GetTodayChallenge)
		protected.POST("/daily-challenge/submit", handlers.SubmitDailyChallenge)
		protected.POST("/daily-challenge/reveal", handlers.RevealDailyChallenge)
		protected.GET("/daily-challenge/leaderboard", handlers.GetDailyChallengeLeaderboard)
		protected.GET("/daily-challenge/practice", handlers.GetDailyChallengePractice)

		// Live Classes (student)
		protected.GET("/live/classes", handlers.ListActiveLiveClasses)
		protected.GET("/live/classes/:id", handlers.GetLiveClass)

		// Push token registration
		protected.POST("/users/push-token", handlers.RegisterPushToken)

		// Doubts
		protected.GET("/doubts", handlers.ListDoubts)
		protected.POST("/doubts", handlers.PostDoubt)
		protected.PUT("/doubts/:id", handlers.UpdateDoubt)
		protected.DELETE("/doubts/:id", handlers.DeleteDoubt)
		protected.GET("/doubts/:id/answers", handlers.GetDoubtAnswers)
		protected.POST("/doubts/:id/answers", handlers.PostDoubtAnswer)

		// Analytics
		protected.GET("/analytics", handlers.GetStudentAnalytics)

		// Bookmarks
		protected.POST("/bookmarks", handlers.AddBookmark)
		protected.DELETE("/bookmarks/:questionId", handlers.RemoveBookmark)
		protected.GET("/bookmarks", handlers.ListBookmarks)
	}

	// WebSocket — auth via query params (no JWT middleware)
	r.GET("/ws/live/:id", handlers.LiveClassWS)
}
