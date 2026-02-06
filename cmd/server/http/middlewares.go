package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the JWT token in Authorization header
func (h *Handlers) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header is missing"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims, err := h.tm.Validate(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		if claims.TokenType != "access" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
			c.Abort()
			return
		}

		// Set userID in context for subsequent handlers
		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Next()
	}
}

// GetUserID retrieves the userID from the context
func GetUserID(c *gin.Context) string {
	return c.GetString("userID")
}

// GetUserEmail retrieves the userEmail from the context
func GetUserEmail(c *gin.Context) string {
	return c.GetString("userEmail")
}
