package models

import (
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	Client      *mongo.Client
	Db          *mongo.Database
	MessagesCol *mongo.Collection
	UsersCol    *mongo.Collection
)
