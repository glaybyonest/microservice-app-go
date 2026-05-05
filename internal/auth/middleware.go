package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}
		fields := strings.Fields(authHeader)
		if len(fields) != 2 || fields[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}
		tokenStr := fields[1]
		claims, err := ParseAccessToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		c.Set("userClaims", claims)
		c.Next()
	}
}

// RoleRequired проверяет, что роль пользователя входит в допустимые роли.
func RoleRequired(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("userClaims")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "No user claims found"})
			return
		}
		userClaims := claims.(*Claims)
		for _, role := range roles {
			if userClaims.Role == role {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	}
}
