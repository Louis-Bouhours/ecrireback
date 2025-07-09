package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Sarinja-Corp/ecrireback/models" // Assurez-vous que le chemin d'import est correct
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv" // <-- Importer la nouvelle bibliothèque
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

// La variable globale est maintenant vide, elle sera chargée depuis le fichier .env
var jwtSecret []byte

// init() est une fonction spéciale en Go qui s'exécute avant main()
// C'est l'endroit parfait pour charger les variables d'environnement.
func init() {
	// Charger le fichier .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erreur lors du chargement du fichier .env")
	}

	// Lire la variable JWT_SECRET depuis l'environnement
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("La variable d'environnement JWT_SECRET n'est pas définie")
	}
	jwtSecret = []byte(secret)
}

func generateJWT(username string, duration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(duration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// La fonction utilise maintenant la variable jwtSecret chargée depuis le .env
	return token.SignedString(jwtSecret)
}

func LoginUserRoutes(r *gin.Engine) {
	r.POST("/api/login", apiUserLogin)
}

// Le reste de votre fonction apiUserLogin ne change pas
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

	isSecure := false

	c.SetCookie("access_token", accessToken, 3600, "/", "localhost", isSecure, true)
	c.SetCookie("refresh_token", refreshToken, 86400, "/", "localhost", isSecure, true)

	c.JSON(http.StatusOK, gin.H{
		"username": user.Username,
		"avatar":   user.Avatar,
	})
}
