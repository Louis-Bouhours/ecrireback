package database

import (
	"context"
	"log"

	"github.com/Sarinja-Corp/ecrireback/internal/models"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connect establishes a connection to MongoDB
func Connect(mongoURI string) (*mongo.Client, error) {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, err
	}

	// Test connection
	if err = client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	log.Println("MongoDB connecté.")
	return client, nil
}

// InitCollections initializes global collection variables
func InitCollections(client *mongo.Client) {
	models.Client = client
	models.UsersCol = client.Database("EcrireDB").Collection("users")
	models.MessagesCol = client.Database("EcrireDB").Collection("messages")
	models.Db = client.Database("EcrireDB")
}

// Disconnect closes the MongoDB connection
func Disconnect(client *mongo.Client) error {
	ctx := context.Background()
	return client.Disconnect(ctx)
}