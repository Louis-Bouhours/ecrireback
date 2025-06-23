package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Message struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Sender    string             `bson:"sender" json:"sender"`
	Receiver  string             `bson:"receiver,omitempty" json:"receiver,omitempty"`
	Content   string             `bson:"content" json:"content"`
	CreatedAt primitive.DateTime `bson:"created_at" json:"created_at"`
}
