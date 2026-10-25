package handlers

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"gateway/database"
	"gateway/entity"
	"github.com/gin-gonic/gin"
)

// OneCTokenMiddleware protects machine-to-machine endpoints. X-1C-Token is
// preferred because it is straightforward to configure in 1C; Bearer is also
// accepted for generic HTTP connector compatibility.
func OneCTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := strings.TrimSpace(os.Getenv("ONEC_INTEGRATION_TOKEN"))
		provided := strings.TrimSpace(c.GetHeader("X-1C-Token"))
		if provided == "" {
			provided = strings.TrimSpace(c.Query("token"))
		}
		if provided == "" {
			provided = strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		}
		valid := expected != "" && len(provided) == len(expected) &&
			subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
		// Generated per-user links use the user's stored token. Keep the
		// environment token as an administrative/backward-compatible option.
		if !valid && c.Query("user_id") != "" {
			var user entity.User
			if database.GetDB() != nil && database.GetDB().Where("id = ? AND onec_token = ?", c.Query("user_id"), provided).First(&user).Error == nil {
				valid = true
			}
		}
		if !valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid 1C integration token"})
			return
		}
		c.Next()
	}
}
