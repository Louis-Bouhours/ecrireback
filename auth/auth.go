package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Louis-Bouhours/ecrireback/db" // Importe notre package database
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NOTE: Pense à charger cette clé depuis une variable d'environnement (.env)
// comme nous en avons discuté pour la sécurité.
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

// GenerateJWT crée un nouveau token JWT signé pour un utilisateur donné.
func GenerateJWT(userID, username string) (string, error) {
	// Définition de la date d'expiration du token (par exemple, 24 heures)
	expirationTime := time.Now().Add(24 * time.Hour)

	// Création des "claims" (les informations contenues dans le token)
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			// ExpiresAt est la date d'expiration au format Unix timestamp
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	// Création du token avec l'algorithme de signature HS256 et les claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Signature du token avec notre clé secrète pour obtenir le string final
	tokenString, err := token.SignedString(JwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWT vérifie la validité d'un token et retourne les claims s'il est valide.
// Cette fonction est nécessaire pour le middleware AuthRequired.
func ValidateJWT(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("algorithme de signature inattendu : %v", token.Header["alg"])
		}
		return JwtKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("token invalide")
	}

	return claims, nil
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

	// Le dernier argument "true" pour HttpOnly rend le cookie inaccessible en JavaScript
	c.SetCookie("token", token, 3600*24, "/", "localhost", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Connecté"})
}

// ==================================================================
// FONCTIONS AJOUTÉES POUR RÉSOUDRE LES ERREURS
// ==================================================================

// AuthRequired est un middleware qui protège les routes.
func AuthRequired(c *gin.Context) {
	// On récupère le token depuis le cookie
	tokenString, err := c.Cookie("token")
	if err != nil {
		// Si pas de cookie, l'utilisateur n'est pas autorisé
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Accès non autorisé"})
		c.Abort() // On arrête le traitement de la requête
		return
	}

	// On valide le token
	claims, err := ValidateJWT(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
		c.Abort()
		return
	}

	// Le token est valide, on peut stocker l'ID utilisateur dans le contexte
	// pour que les handlers suivants puissent l'utiliser
	c.Set("userID", claims.UserID)

	// On passe au handler suivant
	c.Next()
}

// LogoutHandler gère la déconnexion de l'utilisateur.
func LogoutHandler(c *gin.Context) {
	// Pour se déconnecter, on supprime le cookie en lui donnant une date d'expiration passée.
	// MaxAge: -1 demande au navigateur de le supprimer immédiatement.
	c.SetCookie("token", "", -1, "/", "localhost", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Déconnecté"})
}
