package db

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Rdb         *redis.Client
	UsersCol    *mongo.Collection
	MessagesCol *mongo.Collection
	Ctx         = context.Background()
)

func Init() {
	// Charger .env
	if err := godotenv.Load(); err != nil {
		log.Fatal("❌ Erreur : Impossible de charger le fichier .env")
	}

	// Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatal("❌ La variable d'environnement REDIS_ADDR n'est pas définie dans le .env")
	}
	Rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := Rdb.Ping(Ctx).Err(); err != nil {
		log.Fatalf("❌ Impossible de se connecter à Redis: %v", err)
	}
	log.Println("✅ Connecté à Redis")

	// MongoDB
	mongoURI := os.Getenv("BDOUBLED")
	if mongoURI == "" {
		log.Fatal("❌ La variable d'environnement BDOUBLED n'est pas définie dans le .env")
	}

	clientOpts := options.Client().ApplyURI(mongoURI).SetServerSelectionTimeout(10 * time.Second)
	mongoClient, err := mongo.Connect(Ctx, clientOpts)
	if err != nil {
		log.Fatalf("❌ Impossible de se connecter à MongoDB: %v", err)
	}
	db := mongoClient.Database("ecrire_db")
	UsersCol = db.Collection("users")
	MessagesCol = db.Collection("messages")
	RoomsCol = db.Collection("rooms")
	log.Println("✅ Connecté à MongoDB")

	// Index unique sur username
	usersIndex := mongo.IndexModel{
		Keys:    map[string]int{"username": 1},
		Options: options.Index().SetUnique(true).SetName("uniq_username"),
	}
	if _, err := UsersCol.Indexes().CreateOne(Ctx, usersIndex); err != nil {
		log.Printf("⚠️ Impossible de créer l'index unique sur username: %v", err)
	}
}
