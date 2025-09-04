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

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// CORS très permissif côté HTTP (pour WS c’est CheckOrigin dans l’upgrader)
	router.Use(cors.New(cors.Config{
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowWebSockets:  true,
		MaxAge:           12 * time.Hour,
	}))

	// Routes publiques
	router.POST("/api/login", api.ApiUserLogin)
	router.POST("/api/register", api.ApiUserRegister)
	router.GET("/api/me", auth.MeHandler)

	// Logout public (pour le bouton front)
	api.LogoutUserRoutes(router)

	// Routes protégées (exemple)
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

	// Healthcheck simple
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// WebSocket natif
	chat.RegisterWS(router)

	return router
}
