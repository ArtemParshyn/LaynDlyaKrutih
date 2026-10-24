package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// UserIDMiddleware enforces presence of X-User-ID header on all endpoints except /health
func UserIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		header := strings.TrimSpace(c.GetHeader("X-User-ID"))
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "X-User-ID header is required"})
			return
		}

		userID, err := strconv.ParseUint(header, 10, 64)
		if err != nil || userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid X-User-ID header"})
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}
