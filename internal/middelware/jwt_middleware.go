package middelware

import (
	"log"
	"net/http"
	"strings"

	"github.com/AhmadKusumahDEV/go-chat/internal/helpers"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gin-gonic/gin"
)

func JwtAuthMiddleware(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		authHeader := c.GetHeader("Authorization")
		tokenString = authHeader

		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: no token provided"})
			c.Abort()
			return
		}

		claims, err := helpers.GetUserJWT(tokenString, secretKey)
		if err != nil {
			log.Printf("Fail to parse JWT Token: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
			c.Abort()
			return
		}

		userInfo, ok := claims["user_info"].(map[string]any)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: user_info not found in token"})
			c.Abort()
			return
		}

		idUser, ok := claims["user_id"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: user_id not found in token"})
			c.Abort()
			return
		}

		jwtUserInfo := &models.JwtUsersInfo{
			UserID:   userInfo["user_id"].(string),
			Email:    userInfo["email"].(string),
			Username: userInfo["username"].(string),
		}

		c.Set("user_info", jwtUserInfo)
		c.Set("user_id", idUser)
		c.Next()
	}
}
