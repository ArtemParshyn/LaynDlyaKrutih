package handlers

import (
	"crypto/subtle"
	"gateway/database"
	"gateway/entity"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func OneCTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		provided := strings.TrimSpace(c.GetHeader("X-1C-Token"))
		if provided == "" {
			provided = strings.TrimSpace(c.Query("token"))
		}
		expected := strings.TrimSpace(os.Getenv("ONEC_INTEGRATION_TOKEN"))
		valid := expected != "" && len(provided) == len(expected) && subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
		if !valid {
			userID, err := strconv.ParseUint(c.Query("user_id"), 10, 64)
			if err == nil && userID > 0 {
				var user entity.User
				valid = database.GetDB() != nil && database.GetDB().Where("id = ? AND onec_token = ?", userID, provided).First(&user).Error == nil
			}
		}
		if !valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid 1C integration token"})
			return
		}
		c.Next()
	}
}
