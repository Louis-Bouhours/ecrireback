package main

import (
	"log"
	"os"

	"github.com/Louis-Bouhours/ecrireback/chat" // NOUVEL IMPORT
	"github.com/Louis-Bouhours/ecrireback/db"
	"github.com/Louis-Bouhours/ecrireback/routes"
)

func main() {
	// 1. Initialiser la connexion à la base de données
	db.Init()

	// 2. Configurer le routeur Gin principal
	router := routes.SetupRouter()

	// 3. Attacher le serveur Socket.IO au routeur existant
	chat.SetupSocketIO(router)

	// 4. Démarrer le serveur
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8081"
	}
	log.Printf("🚀 Démarrage du serveur sur http://localhost:%s", appPort)
	if err := router.Run(":" + appPort); err != nil {
		log.Fatalf("Erreur lors du démarrage du serveur Gin: %v", err)
	}
}
