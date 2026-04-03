package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"navodaya-api/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := extractClaims(c)
		if err != nil {
			utils.ErrorRes(c, 401, "UNAUTHORIZED", err.Error())
			c.Abort()
			return
		}
		if claims.IsTemp {
			utils.ErrorRes(c, 401, "UNAUTHORIZED", "Full authentication required")
			c.Abort()
			return
		}
		c.Set("userId", claims.UserID)
		c.Set("phone", claims.Phone)
		c.Next()
	}
}

// TrackActivity middleware updates user's last active date and streak
func TrackActivity() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process request first

		// Update activity in background (non-blocking)
		userIDStr, exists := c.Get("userId")
		if !exists {
			return
		}

		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			return
		}

		// Run in goroutine to not block response
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			utils.UpdateUserActivity(ctx, userID)
		}()
	}
}

func RequireTempAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := extractClaims(c)
		if err != nil {
			utils.ErrorRes(c, 401, "UNAUTHORIZED", err.Error())
			c.Abort()
			return
		}
		c.Set("userId", claims.UserID)
		c.Set("phone", claims.Phone)
		c.Set("isTemp", claims.IsTemp)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			utils.ErrorRes(c, 403, "UNAUTHORIZED", "Admin authentication required")
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := utils.ParseAdminToken(token)
		if err != nil {
			utils.ErrorRes(c, 403, "UNAUTHORIZED", "Invalid admin token")
			c.Abort()
			return
		}

		c.Set("adminId", claims.AdminID)
		c.Set("adminEmail", claims.Email)
		c.Set("isSuperAdmin", claims.IsSuperAdmin)
		c.Next()
	}
}

func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			utils.ErrorRes(c, 403, "UNAUTHORIZED", "Admin authentication required")
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := utils.ParseAdminToken(token)
		if err != nil {
			utils.ErrorRes(c, 403, "UNAUTHORIZED", "Invalid admin token")
			c.Abort()
			return
		}

		if !claims.IsSuperAdmin {
			utils.ErrorRes(c, 403, "FORBIDDEN", "Super admin access required")
			c.Abort()
			return
		}

		c.Set("adminId", claims.AdminID)
		c.Set("adminEmail", claims.Email)
		c.Set("isSuperAdmin", claims.IsSuperAdmin)
		c.Next()
	}
}

func extractClaims(c *gin.Context) (*utils.Claims, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("authentication required")
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	return utils.ParseToken(token)
}
