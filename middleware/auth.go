package middleware

import (
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"navodaya-api/utils"
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
		secret := os.Getenv("ADMIN_SECRET")
		if secret == "" || c.GetHeader("X-Admin-Key") != secret {
			utils.ErrorRes(c, 403, "FORBIDDEN", "Admin access required")
			c.Abort()
			return
		}
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
