package jwt

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func JWTAuthMiddleware(service *JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.GetHeader("X-Peer-Token")
		}

		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing authorization token",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			token = authHeader
		}

		claims, err := service.ValidateToken(token)
		if err != nil {
			status := http.StatusUnauthorized
			message := "Invalid token"
			if err == ErrExpiredToken {
				message = "Token expired"
			}
			c.AbortWithStatusJSON(status, gin.H{
				"error": message,
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("roles", claims.Roles)
		c.Set("api_keys", claims.APIKeys)
		c.Set("node_id", claims.NodeID)
		c.Set("is_admin", service.IsAdmin(claims))

		c.Next()
	}
}

func OptionalJWTAuth(service *JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.GetHeader("X-Peer-Token")
		}

		if authHeader == "" {
			c.Next()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := service.ValidateToken(token)
		if err != nil {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("roles", claims.Roles)
		c.Set("api_keys", claims.APIKeys)
		c.Set("node_id", claims.NodeID)
		c.Set("is_admin", service.IsAdmin(claims))

		c.Next()
	}
}

func RequireRole(service *JWTService, roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "No roles found",
			})
			return
		}

		userRoles, ok := claimsVal.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Invalid roles",
			})
			return
		}

		for _, required := range roles {
			for _, userRole := range userRoles {
				if userRole == required {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Insufficient permissions",
		})
	}
}

func GetUserID(c *gin.Context) string {
	if val, ok := c.Get("user_id"); ok {
		return val.(string)
	}
	return ""
}

func GetRoles(c *gin.Context) []string {
	if val, ok := c.Get("roles"); ok {
		if roles, ok := val.([]string); ok {
			return roles
		}
	}
	return nil
}

func GetNodeID(c *gin.Context) string {
	if val, ok := c.Get("node_id"); ok {
		return val.(string)
	}
	return ""
}

func IsAdmin(c *gin.Context) bool {
	if val, ok := c.Get("is_admin"); ok {
		return val.(bool)
	}
	return false
}
