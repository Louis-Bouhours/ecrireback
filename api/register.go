package api

import (
	_ "context"
	"net/http"

	"github.com/Louis-Bouhours/ecrireback/db"
	"github.com/Louis-Bouhours/ecrireback/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

func ApiUserRegister(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Avatar   string `json:"avatar"`
	}
	if err := c.BindJSON(&body); err != nil || body.Username == "" || body.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload invalide"})
		return
	}

	// Garde de sécurité si la collection n'est pas initialisée
	if db.UsersCol == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Base non initialisée"})
		return
	}

	// Vérifier l’unicité du username
	count, err := db.UsersCol.CountDocuments(c, bson.M{"username": body.Username})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur base"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nom déjà pris"})
		return
	}

	// Hash du mot de passe
	hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}

	newUser := models.User{
		Username: body.Username,
		Password: string(hashed),
		Avatar:   body.Avatar,
	}

	_, err = db.UsersCol.InsertOne(c, newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création utilisateur"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"username": newUser.Username, "avatar": newUser.Avatar})
}
