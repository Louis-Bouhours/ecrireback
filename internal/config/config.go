package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	JWTSecret      string
	MongoURI       string
	RedisAddr      string
	RedisPassword  string
	ServerPort     string
	AllowedOrigins []string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	// Load .env file
	err := godotenv.Load("configs/.env")
	if err != nil {
		log.Println("Warning: Could not load .env file, using environment variables")
	}

	return &Config{
		JWTSecret:      getEnv("JWT_SECRET", "default_secret_change_in_production"),
		MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017/ecriredb?authSource=admin"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		ServerPort:     getEnv("SERVER_PORT", ":8080"),
		AllowedOrigins: []string{getEnv("ALLOWED_ORIGINS", "http://localhost:3000")},
	}
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}