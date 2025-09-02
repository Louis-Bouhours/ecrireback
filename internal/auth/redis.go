package auth

import (
	"context"
	"github.com/redis/go-redis/v9"
	"log"
)

var Rdb *redis.Client
var Ctx = context.Background()

func InitRedis() {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // adapte si Docker ou distant
		Password: "",               // ou ton mot de passe Redis
		DB:       0,
	})
	_, err := Rdb.Ping(Ctx).Result()
	if err != nil {
		log.Fatalf("Erreur connexion Redis : %v", err)
	}
}
