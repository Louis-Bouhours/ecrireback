package routes

import (
	"net/http"

	"github.com/Louis-Bouhours/ecrireback/api"
	"github.com/Louis-Bouhours/ecrireback/auth" // Importe notre package auth
	"github.com/gin-gonic/gin"
)

// SetupRouter configure et retourne le routeur Gin.
func SetupRouter() *gin.Engine {
	router := gin.Default()

	// Routes publiques
	router.POST("/login", auth.LoginHandler)
	router.POST("/chat/token", api.ChatJoinToken)

	// Routes protégées
	authorized := router.Group("/")
	authorized.Use(auth.AuthRequired)
	{
		authorized.POST("/logout", auth.LogoutHandler)
		authorized.GET("/profile", func(c *gin.Context) {
			userID := c.MustGet("userID").(string)
			username := c.MustGet("username").(string)
			c.JSON(http.StatusOK, gin.H{
				"message":  "Ceci est une route protégée",
				"userID":   userID,
				"username": username,
			})
		})
	}

	return router
}
