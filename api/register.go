package api

import (
	"net/http"
	"time"

	"github.com/Louis-Bouhours/ecrireback/db"
	"github.com/Louis-Bouhours/ecrireback/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive" // Importation nécessaire pour l'ObjectId
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

	if db.UsersCol == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Base non initialisée"})
		return
	}

	// Vérifier l'unicité du username
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

	// Création du nouvel utilisateur
	newUser := models.User{
		Username: body.Username,
		Password: string(hashed),
		Avatar:   body.Avatar,
	}

	// Insertion dans la base de données
	result, err := db.UsersCol.InsertOne(c, newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création utilisateur"})
		return
	}

	// Récupération de l'ID inséré
	// La méthode InsertOne retourne l'ID du document créé.
	insertedID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération de l'ID"})
		return
	}

	accessToken, err := generateJWT(newUser.Username, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur interne à la génération du token"})
		return
	}

	refreshToken, err := generateJWT(newUser.Username, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur interne à la génération du token"})
		return
	}

	isSecure := false

	c.SetCookie("access_token", accessToken, 3600, "/", "localhost", isSecure, true)
	c.SetCookie("refresh_token", refreshToken, 86400, "/", "localhost", isSecure, true)

	// Le champ ID de newUser n'est pas rempli ici, donc on le renvoie via insertedID
	c.JSON(http.StatusOK, gin.H{"username": newUser.Username, "avatar": newUser.Avatar, "id": insertedID.Hex()})
}
