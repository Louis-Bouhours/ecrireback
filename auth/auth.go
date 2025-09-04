package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Louis-Bouhours/ecrireback/db"
	"github.com/Louis-Bouhours/ecrireback/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var (
	cfg     config
	cfgOnce sync.Once
)

type config struct {
	jwtKey          []byte
	cookieDomain    string
	cookieSecure    bool
	cookieSameSite  http.SameSite
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func ensureConfig() {
	cfgOnce.Do(func() {
		secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
		if secret == "" {
			panic("JWT_SECRET must be set in environment")
		}
		cfg.jwtKey = []byte(secret)

		domain := os.Getenv("COOKIE_DOMAIN")
		if domain == "" {
			domain = "localhost"
		}
		cfg.cookieDomain = domain

		secureEnv := strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SECURE")))
		cfg.cookieSecure = secureEnv == "1" || secureEnv == "true" || secureEnv == "yes"

		switch strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SAMESITE"))) {
		case "none":
			cfg.cookieSameSite = http.SameSiteNoneMode
		case "strict":
			cfg.cookieSameSite = http.SameSiteStrictMode
		default:
			cfg.cookieSameSite = http.SameSiteLaxMode
		}

		cfg.accessTokenTTL = parseDurationDefault(os.Getenv("ACCESS_TOKEN_TTL"), time.Hour)
		cfg.refreshTokenTTL = parseDurationDefault(os.Getenv("REFRESH_TOKEN_TTL"), 24*time.Hour)
	})
}

func parseDurationDefault(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	if secs, err := strconv.Atoi(s); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return def
}

type Claims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	TokenType string `json:"token_type"` // "access" | "refresh"
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Identifier string `json:"identifier"` // username or email
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Avatar   string `json:"avatar"`
	Email    string `json:"email"` // optionnel mais recommandé
}

func generateJWT(userID, username, tokenType string, ttl time.Duration) (string, error) {
	ensureConfig()
	now := time.Now()
	claims := &Claims{
		UserID:    userID,
		Username:  username,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(cfg.jwtKey)
}

func ValidateJWT(tokenStr string) (*Claims, error) {
	ensureConfig()
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("algorithme de signature inattendu: %v", t.Header["alg"])
		}
		return cfg.jwtKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token invalide")
	}
	return claims, nil
}

func setCookie(w http.ResponseWriter, name, value string, maxAge int) {
	ensureConfig()
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   cfg.cookieDomain,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
		Secure:   cfg.cookieSecure,
		HttpOnly: true,
		SameSite: cfg.cookieSameSite,
	})
}

func clearCookie(w http.ResponseWriter, name string) {
	ensureConfig()
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   cfg.cookieDomain,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Secure:   cfg.cookieSecure,
		HttpOnly: true,
		SameSite: cfg.cookieSameSite,
	})
}

func setAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	setCookie(c.Writer, "access_token", accessToken, int(cfg.accessTokenTTL.Seconds()))
	setCookie(c.Writer, "refresh_token", refreshToken, int(cfg.refreshTokenTTL.Seconds()))
}

func clearAuthCookies(c *gin.Context) {
	clearCookie(c.Writer, "access_token")
	clearCookie(c.Writer, "refresh_token")
}

// LoginHandler: accepte identifier OR username/email + password.
// Cherche par username OU email (email normalisé en lower), compare bcrypt, émet access+refresh en cookies.
func LoginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Password) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Requête invalide"})
		return
	}

	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		if req.Username != "" {
			identifier = req.Username
		} else if req.Email != "" {
			identifier = req.Email
		}
	}
	if identifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Identifiant requis"})
		return
	}

	var user models.User
	filter := bson.M{
		"$or": bson.A{
			bson.M{"username": identifier},
			bson.M{"email": strings.ToLower(identifier)},
		},
	}
	if err := db.UsersCol.FindOne(context.TODO(), filter).Decode(&user); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants incorrects"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants incorrects"})
		return
	}

	uid := user.ID.Hex()
	access, err := generateJWT(uid, user.Username, "access", cfg.accessTokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
		return
	}
	refresh, err := generateJWT(uid, user.Username, "refresh", cfg.refreshTokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
		return
	}

	setAuthCookies(c, access, refresh)
	c.JSON(http.StatusOK, gin.H{
		"id":       uid,
		"username": user.Username,
		"email":    user.Email,
		"avatar":   user.Avatar,
	})
}

// RegisterHandler: crée un utilisateur (hash bcrypt), émet les cookies.
func RegisterHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil ||
		strings.TrimSpace(req.Username) == "" ||
		strings.TrimSpace(req.Password) == "" ||
		strings.TrimSpace(req.Email) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload invalide"})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Unicité
	if n, err := db.UsersCol.CountDocuments(c, bson.M{"username": req.Username}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur base (username)"})
		return
	} else if n > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nom déjà pris"})
		return
	}
	if n, err := db.UsersCol.CountDocuments(c, bson.M{"email": req.Email}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur base (email)"})
		return
	} else if n > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email déjà utilisé"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}

	newUser := models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashed),
		Avatar:   req.Avatar,
	}
	result, err := db.UsersCol.InsertOne(c, newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création utilisateur"})
		return
	}

	oid, _ := result.InsertedID.(primitive.ObjectID)
	uid := oid.Hex()

	access, err := generateJWT(uid, newUser.Username, "access", cfg.accessTokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
		return
	}
	refresh, err := generateJWT(uid, newUser.Username, "refresh", cfg.refreshTokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
		return
	}

	setAuthCookies(c, access, refresh)
	c.JSON(http.StatusOK, gin.H{
		"id":       uid,
		"username": newUser.Username,
		"email":    newUser.Email,
		"avatar":   newUser.Avatar,
	})
}

func RefreshHandler(c *gin.Context) {
	rt, err := c.Cookie("refresh_token")
	if err != nil || strings.TrimSpace(rt) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}
	claims, err := ValidateJWT(rt)
	if err != nil || claims.TokenType != "refresh" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
		return
	}

	access, err := generateJWT(claims.UserID, claims.Username, "access", cfg.accessTokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
		return
	}
	newRefresh, err := generateJWT(claims.UserID, claims.Username, "refresh", cfg.refreshTokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
		return
	}

	setAuthCookies(c, access, newRefresh)
	c.JSON(http.StatusOK, gin.H{"message": "Tokens renouvelés"})
}

func AuthRequired(c *gin.Context) {
	at, err := c.Cookie("access_token")
	if err != nil || strings.TrimSpace(at) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Accès non autorisé"})
		c.Abort()
		return
	}
	claims, err := ValidateJWT(at)
	if err != nil || claims.TokenType != "access" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
		c.Abort()
		return
	}
	c.Set("userID", claims.UserID)
	c.Set("username", claims.Username)
	c.Next()
}

func LogoutHandler(c *gin.Context) {
	clearAuthCookies(c)
	c.JSON(http.StatusOK, gin.H{"message": "Déconnecté"})
}
