package db

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv" // N'oubliez pas l'import
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Rdb         *redis.Client
	UsersCol    *mongo.Collection
	Ctx         = context.Background()
	MessagesCol *mongo.Collection
)

// Init charge les variables d'environnement et se connecte aux bases de données.
func Init() {
	// ÉTAPE 1 : Charger le fichier .env AU DÉBUT.
	if err := godotenv.Load(); err != nil {
		log.Fatal("❌ Erreur : Impossible de charger le fichier .env")
	}

	// --- Connexion à Redis ---
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatal("❌ La variable d'environnement REDIS_ADDR n'est pas définie dans le .env")
	}

	Rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := Rdb.Ping(Ctx).Err(); err != nil {
		log.Fatalf("❌ Impossible de se connecter à Redis: %v", err)
	}
	log.Println("✅ Connecté à Redis")

	// --- Connexion à MongoDB ---
	mongoURI := os.Getenv("BDOUBLED") // On utilise le nom de votre variable
	if mongoURI == "" {
		log.Fatal("❌ La variable d'environnement BDOUBLED n'est pas définie dans le .env")
	}

	mongoClient, err := mongo.Connect(Ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("❌ Impossible de se connecter à MongoDB: %v", err)
	}
	UsersCol = mongoClient.Database("ecrire_db").Collection("users")
	MessagesCol = mongoClient.Database("ecrire_db").Collection("messages")
	log.Println("✅ Connecté à MongoDB")
}
