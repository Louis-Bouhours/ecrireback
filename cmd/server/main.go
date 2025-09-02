package main

import (
	"context"
	"log"

	"github.com/Sarinja-Corp/ecrireback/internal/api/handlers"
	"github.com/Sarinja-Corp/ecrireback/internal/auth"
	"github.com/Sarinja-Corp/ecrireback/internal/chat"
	"github.com/Sarinja-Corp/ecrireback/internal/models"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	auth.InitRedis()
	r := gin.Default()

	// 1. Appliquer les middlewares généraux (comme CORS) en premier
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	// 2. Établir la connexion à la base de données AVANT de définir les routes
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017/ecriredb?authSource=admin"))
	if err != nil {
		log.Fatalf("Erreur MongoDB: %v", err)
	}
	defer func(client *mongo.Client, ctx context.Context) {
		if err := client.Disconnect(ctx); err != nil {
			log.Fatalf("Erreur déconnexion MongoDB: %v", err)
		}
	}(client, ctx)

	if err = client.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB indisponible!")
	}
	log.Println("MongoDB connecté.")

	// Assigner la connexion à la variable globale
	models.Client = client
	models.UsersCol = client.Database("EcrireDB").Collection("users")

	// 3. Maintenant que la BDD est connectée, on peut enregistrer les routes
	chat.SetupSocketIO(r)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "YOUPY"})
	})
	r.POST("/login", auth.LoginHandler) // Note: Assurez-vous que ce handler n'a pas été déplacé dans les routes API
	r.POST("/logout", auth.LogoutHandler)
	r.Static("/static", "./static")

	r.GET("/profile", auth.AuthRequired(), func(c *gin.Context) {
		username := c.GetString("username")
		c.JSON(200, gin.H{"message": "Bienvenue " + username})
	})

	// Inclusion des routes API qui dépendent de la BDD
	handlers.RegisterUserRoutes(r)
	handlers.LoginUserRoutes(r)
	handlers.LogoutUserRoutes(r)

	// 4. Démarrer le serveur
	log.Println("Serveur sur http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Erreur serveur: %v", err)
	}
}
