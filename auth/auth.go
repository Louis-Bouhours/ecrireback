package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// =================================================================
// SECTION: Configuration Globale et Initialisation
// =================================================================

var (
	// Rdb is the client for the Redis connection.
	Rdb *redis.Client
	// Ctx is the global context for database operations.
	Ctx = context.Background()
	// UsersCol is the MongoDB users collection.
	UsersCol *mongo.Collection
	// JwtKey is the secret key for signing JWTs.
	// ⚠️ IMPORTANT: In a real application, get this from an environment variable!
	JwtKey = []byte("VOTRE_CLE_SECRETE_ULTRA_SECURISEE")
)

// init() is executed automatically when the program starts.
// It's the perfect place to initialize database connections.
func init() {
	// --- Redis Initialization for Docker ---
	// Get the Redis address from an environment variable, defaulting to the Docker service name.
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379" // 'redis' is the service name in docker-compose.yml
	}

	Rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Verify the Redis connection.
	if err := Rdb.Ping(Ctx).Err(); err != nil {
		log.Fatalf("❌ Could not connect to Redis at %s: %v", redisAddr, err)
	}
	log.Printf("✅ Successfully connected to Redis at %s", redisAddr)

	// --- MongoDB Initialization for Docker ---
	// Get the MongoDB URI from an environment variable, defaulting to the Docker service name.
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://mongo:27017" // 'mongo' is the service name in docker-compose.yml
	}

	clientOptions := options.Client().ApplyURI(mongoURI)
	mongoClient, err := mongo.Connect(Ctx, clientOptions)
	if err != nil {
		log.Fatalf("❌ Could not connect to MongoDB at %s: %v", mongoURI, err)
	}

	// Verify the MongoDB connection.
	if err := mongoClient.Ping(Ctx, nil); err != nil {
		log.Fatalf("❌ Could not ping MongoDB at %s: %v", mongoURI, err)
	}

	UsersCol = mongoClient.Database("ecrire_db").Collection("users")
	log.Printf("✅ Successfully connected to MongoDB at %s", mongoURI)
}

// =================================================================
// SECTION: JWT Logic
// =================================================================

// Claims defines the structure of the data inside the JWT.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a new JWT for a user.
func GenerateJWT(userID, username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtKey)
}

// ValidateJWT verifies the validity of a JWT.
func ValidateJWT(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return JwtKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

// =================================================================
// SECTION: Authentication Handlers
// =================================================================

// LoginRequest is the expected structure for a login request.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginHandler handles user login.
func LoginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request, username and password are required"})
		return
	}

	var user bson.M
	err := UsersCol.FindOne(Ctx, bson.M{"username": req.Username, "password": req.Password}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect credentials"})
		return
	}

	// MongoDB's default _id is an ObjectID, not a string. We must convert it.
	userID, ok := user["_id"].(primitive.ObjectID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format in database"})
		return
	}

	token, err := GenerateJWT(userID.Hex(), req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Set the token in an HttpOnly cookie for security.
	// The domain is left empty to default to the origin.
	c.SetCookie("token", token, 3600*24, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged in"})
}

// LogoutHandler handles user logout by blacklisting the token in Redis.
func LogoutHandler(c *gin.Context) {
	tokenStr, err := c.Cookie("token")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Already logged out"})
		return
	}

	claims, err := ValidateJWT(tokenStr)
	if err == nil {
		// Add the token to a "blacklist" in Redis until it naturally expires.
		expiration := time.Until(claims.ExpiresAt.Time)
		if expiration > 0 {
			Rdb.Set(Ctx, "blacklist:"+tokenStr, "true", expiration)
		}
	}

	// Clear the client-side cookie.
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

// =================================================================
// SECTION: Authentication Middleware
// =================================================================

// AuthRequired is a middleware to protect routes.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := c.Cookie("token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		// Check if the token is in the Redis blacklist.
		val, err := Rdb.Get(Ctx, "blacklist:"+tokenStr).Result()
		if err == nil && val == "true" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token (session has been logged out)"})
			return
		}

		// Validate the JWT.
		claims, err := ValidateJWT(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Add user information to the context for subsequent handlers.
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// =================================================================
// SECTION: Application Entrypoint
// =================================================================

func main() {
	router := gin.Default()

	// Public routes
	router.POST("/login", LoginHandler)

	// Protected routes using the middleware
	authorized := router.Group("/")
	authorized.Use(AuthRequired())
	{
		authorized.POST("/logout", LogoutHandler)
		authorized.GET("/profile", func(c *gin.Context) {
			// We can retrieve user info thanks to the middleware
			userID := c.MustGet("userID").(string)
			username := c.MustGet("username").(string)
			c.JSON(http.StatusOK, gin.H{
				"message":  "This is a protected route",
				"userID":   userID,
				"username": username,
			})
		})
	}

	// The port matches the one in your docker-compose.yml
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8081"
	}

	log.Printf("🚀 Starting server on http://localhost:%s", appPort)
	if err := router.Run(":" + appPort); err != nil {
		log.Fatalf("Failed to run Gin server: %v", err)
	}
}
