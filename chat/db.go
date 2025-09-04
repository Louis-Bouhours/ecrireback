package chat

import (
	"context"
	"log"
	"time"

	"github.com/Louis-Bouhours/ecrireback/db"
	"github.com/Louis-Bouhours/ecrireback/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SaveMessage sauvegarde un message dans MongoDB de façon asynchrone
// SaveMessage sauvegarde un message dans MongoDB de façon asynchrone
func SaveMessage(msg WSMessage, userID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Conversion en modèle de message pour MongoDB
		chatMsg := models.ChatMessage{
			Username:  msg.Username,
			Text:      msg.Text,
			Timestamp: msg.Timestamp,
			Room:      msg.Room,
			UserID:    userID,
		}

		// Utilisation de MessagesCol au lieu de db.DB.Collection("messages")
		result, err := db.MessagesCol.InsertOne(ctx, chatMsg)
		if err != nil {
			log.Printf("Erreur lors de la sauvegarde du message: %v", err)
			return
		}

		log.Printf("Message sauvegardé avec ID: %v", result.InsertedID)
	}()
}

// GetHistoricalMessages récupère les messages historiques d'une room
func GetHistoricalMessages(room string) []WSMessage {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Options pour trier par timestamp et limiter le nombre de messages
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: 1}}).
		SetLimit(100) // Limiter à 100 derniers messages pour éviter de surcharger

	// Utilisation de MessagesCol au lieu de db.DB.Collection("messages")
	cursor, err := db.MessagesCol.Find(ctx, bson.M{"room": room}, opts)
	if err != nil {
		log.Printf("Erreur lors de la récupération des messages historiques: %v", err)
		return nil
	}
	defer cursor.Close(ctx)

	var messages []models.ChatMessage
	if err = cursor.All(ctx, &messages); err != nil {
		log.Printf("Erreur lors du décodage des messages: %v", err)
		return nil
	}

	// Conversion en WSMessage
	wsMessages := make([]WSMessage, len(messages))
	for i, msg := range messages {
		wsMessages[i] = WSMessage{
			Username:  msg.Username,
			Text:      msg.Text,
			Timestamp: msg.Timestamp,
			Room:      msg.Room,
		}
	}

	return wsMessages
}
