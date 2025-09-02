package main

import (
	"log"
	"os"

	"github.com/Louis-Bouhours/ecrireback/db"
	"github.com/Louis-Bouhours/ecrireback/routes"
)

func main() {
	// 1. Initialiser la connexion à la base de données
	db.Init()

	// 2. Configurer le routeur
	router := routes.SetupRouter()

	// 3. Démarrer le serveur
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8081"
	}
	log.Printf("🚀 Démarrage du serveur sur http://localhost:%s", appPort)
	if err := router.Run(":" + appPort); err != nil {
		log.Fatalf("Erreur lors du démarrage du serveur Gin: %v", err)
	}
}
