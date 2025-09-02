package db

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Rdb      *redis.Client
	UsersCol *mongo.Collection
	Ctx      = context.Background()
)

// Init se connecte à Redis et MongoDB.
func Init() {
	// --- Connexion à Redis ---
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	Rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := Rdb.Ping(Ctx).Err(); err != nil {
		log.Fatalf("❌ Impossible de se connecter à Redis: %v", err)
	}
	log.Println("✅ Connecté à Redis")

	// --- Connexion à MongoDB ---
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://mongo:27017"
	}
	mongoClient, err := mongo.Connect(Ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("❌ Impossible de se connecter à MongoDB: %v", err)
	}
	UsersCol = mongoClient.Database("ecrire_db").Collection("users")
	log.Println("✅ Connecté à MongoDB")
}
