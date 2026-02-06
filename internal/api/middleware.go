package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth validates the API key from either header or query parameter
func APIKeyAuth(validKeys []string) gin.HandlerFunc {
	keyMap := make(map[string]bool)
	for _, key := range validKeys {
		keyMap[key] = true
	}

	return func(c *gin.Context) {
		// Try to get key from header first
		apiKey := c.GetHeader("X-API-Key")

		// Fall back to query parameter
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		// Fall back to Authorization header with Bearer scheme
		if apiKey == "" {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "API key is required (use X-API-Key header, api_key query param, or Bearer token)",
			})
			c.Abort()
			return
		}

		if !keyMap[apiKey] {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
