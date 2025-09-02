package auth

import (
	"github.com/Sarinja-Corp/ecrireback/db" // Importe notre package database
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"time"
)

var JwtKey = []byte("JWT_SECRET")

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// ... (GenerateJWT et ValidateJWT restent ici, sans changement)

func GenerateJWT(userID, username string) (string, error) {
	// ...
}

func ValidateJWT(tokenStr string) (*Claims, error) {
	// ...
}

func LoginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Requête invalide"})
		return
	}

	var user bson.M
	// Utilise les variables exportées du package database
	err := db.UsersCol.FindOne(db.Ctx, bson.M{"username": req.Username, "password": req.Password}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants incorrects"})
		return
	}

	userID, ok := user["_id"].(primitive.ObjectID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Format ID invalide"})
		return
	}

	token, err := GenerateJWT(userID.Hex(), req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur de token"})
		return
	}

	c.SetCookie("token", token, 3600*24, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Connecté"})
}

// ... (LogoutHandler et AuthRequired restent ici, en utilisant database.Rdb et database.Ctx)

func LogoutHandler(c *gin.Context) {
	// ...
}

func AuthRequired() gin.HandlerFunc {
	// ...
}
