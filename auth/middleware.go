package auth

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentification requise"})
			return
		}

		// Vérifie blacklist
		exists, _ := Rdb.Get(Ctx, "bl:"+token).Result()
		if exists == "true" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
			return
		}

		claims, err := ValidateJWT(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
