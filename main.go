package main

import (
	"context"
	"github.com/Sarinja-Corp/ecrireback/api"
	"github.com/Sarinja-Corp/ecrireback/auth"
	"github.com/Sarinja-Corp/ecrireback/chat"
	"github.com/Sarinja-Corp/ecrireback/models"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

func main() {
	auth.InitRedis()
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	chat.SetupSocketIO(router)

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "YOUPY",
		})
	})
	r.POST("/login", auth.LoginHandler)
	r.POST("/logout", auth.LogoutHandler)
	r.Static("/static", "./static")

	r.GET("/profile", auth.AuthRequired(), func(c *gin.Context) {
		username := c.GetString("username")
		c.JSON(200, gin.H{"message": "Bienvenue " + username})
	})

	ctx := context.Background()
	var err error
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017/ecriredb?authSource=admin"))
	if err != nil {
		log.Fatalf("Erreur MongoDB: %v", err)
	}
	defer func(client *mongo.Client, ctx context.Context) {
		err := client.Disconnect(ctx)
		if err != nil {
			log.Fatalf("Erreur déconnexion MongoDB: %v", err)
		}
	}(client, ctx)
	if err = client.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB indisponible!")
	}
	log.Println("MongoDB connecté.")

	models.Client = client
	models.UsersCol = client.Database("EcrireDB").Collection("users")

	// Inclusion des routes API
	api.RegisterUserRoutes(r)
	api.LoginUserRoutes(r)
	api.LogoutUserRoutes(r)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allows all origins (for development only!)
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}))
	log.Println("Serveur sur http://localhost:8080")
	err = r.Run(":8080")
	if err != nil {
		log.Fatalf("Erreur serveur: %v", err)
	}
}
