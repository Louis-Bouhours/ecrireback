package auth

import (
	"context"
	"github.com/Sarinja-Corp/Ecrire/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"time"
	_ "time"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Requête invalide"})
		return
	}

	var user bson.M
	err := models.UsersCol.FindOne(context.TODO(), bson.M{"username": req.Username, "password": req.Password}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants incorrects"})
		return
	}

	userID := user["_id"].(interface{})

	token, err := GenerateJWT(userID.(string), req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération JWT"})
		return
	}

	c.SetCookie("token", token, 3600*24, "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{"message": "Connecté avec succès"})
}

func LogoutHandler(c *gin.Context) {
	tokenStr, err := c.Cookie("token")
	if err == nil {
		// on stocke le token comme clé dans Redis
		claims, err := ValidateJWT(tokenStr)
		if err == nil {
			exp := time.Until(claims.ExpiresAt.Time)
			Rdb.Set(Ctx, "bl:"+tokenStr, "true", exp)
		}
	}
	// Supprimer cookie client
	c.SetCookie("token", "", -1, "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{"message": "Déconnecté"})
}
