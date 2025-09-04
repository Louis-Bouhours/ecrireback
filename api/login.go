package api

import (
	"context"
	"net/http"

	"github.com/Louis-Bouhours/ecrireback/auth"
	"github.com/Louis-Bouhours/ecrireback/db"
	"github.com/Louis-Bouhours/ecrireback/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func ApiUserLogin(c *gin.Context) { auth.LoginHandler(c) }

func ApiMe(c *gin.Context) {
	tokenStr, err := c.Cookie("access_token")
	if err != nil || tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}
	claims, err := auth.ValidateJWT(tokenStr)
	if err != nil || claims.TokenType != "access" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
		return
	}

	var user models.User
	_ = db.UsersCol.FindOne(context.TODO(), bson.M{"username": claims.Username}).Decode(&user)

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID.Hex(),
		"username": claims.Username,
		"email":    user.Email,
		"avatar":   user.Avatar,
	})
}
