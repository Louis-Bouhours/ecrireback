package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func LogoutUserRoutes(r *gin.Engine) {
	r.POST("/api/logout", apiUserLogout)
}

func apiUserLogout(c *gin.Context) {
	// Ici, vous pouvez gérer la logique de déconnexion si nécessaire
	// Par exemple, supprimer le token d'authentification ou gérer la session.
	c.JSON(http.StatusOK, gin.H{"message": "Déconnexion réussie"})
}
