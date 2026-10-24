package handlers

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

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
			provided = strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		}
		valid := expected != "" && len(provided) == len(expected) &&
			subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
		if !valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid 1C integration token"})
			return
		}
		c.Next()
	}
}
