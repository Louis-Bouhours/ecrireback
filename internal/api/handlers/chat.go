package handlers

import (
	"fmt"
	"github.com/Sarinja-Corp/ecrireback/internal/auth"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"net/http"
	"strings"
	"time"
)

// Simple map en mémoire pour les pseudos actifs (non persistant)
var onlineUsers = make(map[string]struct{})

func ChatRoutes(r *gin.Engine) {
	server := socketio.NewServer(nil)

	server.OnConnect("/", func(s socketio.Conn) error {
		fmt.Println("Nouveau client connecté:", s.ID())
		return nil
	})

	server.OnEvent("/", "join", func(s socketio.Conn, token string) {
		claims, err := auth.ValidateJWT(token)
		if err != nil {
			s.Emit("error", "Token invalide")
			err := s.Close()
			if err != nil {
				return
			}
			return
		}

		username := claims.Username
		if username == "" {
			s.Emit("error", "Pseudo manquant")
			err := s.Close()
			if err != nil {
				return
			}
			return
		}
		onlineUsers[username] = struct{}{}
		s.SetContext(username)
		s.Emit("joined", username)
		server.BroadcastToNamespace("/", "user_list", userList())
	})

	server.OnEvent("/", "message", func(s socketio.Conn, msg string) {
		username := s.Context().(string)
		if strings.TrimSpace(msg) == "" {
			return
		}
		now := time.Now().Format("15:04:05")
		message := map[string]string{
			"user":    username,
			"message": msg,
			"heure":   now,
		}
		server.BroadcastToNamespace("/", "message", message)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		username := s.Context()
		if username != nil {
			delete(onlineUsers, username.(string))
			server.BroadcastToNamespace("/", "user_list", userList())
		}
	})

	go func() {
		err := server.Serve()
		if err != nil {

		}
	}()
	r.GET("/socket.io/*any", gin.WrapH(server))
	r.POST("/socket.io/*any", gin.WrapH(server))
}

func userList() []string {
	l := make([]string, 0, len(onlineUsers))
	for u := range onlineUsers {
		l = append(l, u)
	}
	return l
}

// ChatJoinToken Endpoint pour recevoir un pseudo et filer un token JWT simple
func ChatJoinToken(c *gin.Context) {
	var payload struct {
		Username string `json:"username"`
	}
	if err := c.BindJSON(&payload); err != nil || payload.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pseudo requis"})
		return
	}
	// On ne regarde pas la BDD !
	tok, err := auth.GenerateJWT("", payload.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Problème token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "username": payload.Username})
}
