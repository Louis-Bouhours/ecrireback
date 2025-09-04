package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username string             `bson:"username" json:"username"`
	Email    string             `bson:"email,omitempty" json:"email,omitempty"`
	Password string             `bson:"password" json:"-"`
	Avatar   string             `bson:"avatar,omitempty" json:"avatar,omitempty"`
}
