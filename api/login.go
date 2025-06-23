package api

import (
	"context"
	"github.com/Sarinja-Corp/Ecrire/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

var jwtSecret = []byte("TA_SUPER_CLE_SECRETE") // Change ça et mets-la secrète/env variable

func generateJWT(username string, duration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(duration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func LoginUserRoutes(r *gin.Engine) {
	r.POST("/api/login", apiUserLogin)
}

func apiUserLogin(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&body); err != nil || body.Username == "" || body.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Champs requis"})
		return
	}

	var user models.User
	err := models.UsersCol.FindOne(context.TODO(), bson.M{"username": body.Username}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur ou mot de passe invalide"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur ou mot de passe invalide"})
		return
	}

	accessToken, err := generateJWT(user.Username, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur interne à la génération du token"})
		return
	}

	refreshToken, err := generateJWT(user.Username, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur interne à la génération du token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username": user.Username,
		"avatar":   user.Avatar,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		HttpOnly: true,
		Secure:   true, // active en prod HTTPS
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   3600,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})
}
