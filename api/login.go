package api

import (
	"context"
	"net/http"

	"github.com/Louis-Bouhours/ecrireback/db"
	"github.com/Louis-Bouhours/ecrireback/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ApiMe(c *gin.Context) {
	// Récupérer l'ID utilisateur depuis le contexte (mis par le middleware AuthRequired)
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	// Convertir string ID en ObjectID
	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ID utilisateur invalide"})
		return
	}

	// Récupérer l'utilisateur depuis la base de données
	var user models.User
	err = db.UsersCol.FindOne(context.TODO(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Utilisateur non trouvé"})
		return
	}

	// Retourner les informations utilisateur
	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID.Hex(),
		"username": user.Username,
		"email":    user.Email,
		"avatar":   user.Avatar,
	})
}
