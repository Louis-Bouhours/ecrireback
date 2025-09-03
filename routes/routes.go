package routes

import (
	"net/http"
	"time"

	"github.com/Louis-Bouhours/ecrireback/api"
	"github.com/Louis-Bouhours/ecrireback/auth"
	"github.com/Louis-Bouhours/ecrireback/chat"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter configure et retourne le routeur Gin.
func SetupRouter() *gin.Engine {
	router := gin.Default()

	// CORS avec credentials + réflexion de l'origin (ALLOW ALL en dev)
	// Remarque: on n'utilise PAS AllowOrigins: ["*"] si AllowCredentials = true
	router.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// En prod, remplace par une whitelist explicite:
			// return origin == "https://ton-domaine" || origin == "https://app.ton-domaine"
			return true
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,           // IMPORTANT pour cookies/Authorization cross-origin
		MaxAge:           12 * time.Hour, // Cache des préflights
	}))

	// Routes publiques
	router.POST("/api/login", api.ApiUserLogin)
	router.POST("/chat/token", chat.ChatJoinToken)
	router.POST("/api/register", api.ApiUserRegister)

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
