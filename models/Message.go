package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatMessage représente un message stocké dans la base de données
type ChatMessage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username  string             `bson:"username" json:"username"`
	Text      string             `bson:"text" json:"text"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
	Room      string             `bson:"room" json:"room"`
	UserID    string             `bson:"userId,omitempty" json:"userId,omitempty"`
}
