package chat

import (
	"errors"
	"log"
	"net/http"

	"github.com/Louis-Bouhours/ecrireback/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	socketio "github.com/googollee/go-socket.io"
)

func getUsernameFromSocket(s socketio.Conn) (string, error) {
	req := s.RemoteHeader()
	cookieHeader := req.Get("Cookie")
	cookies := (&http.Request{Header: http.Header{"Cookie": {cookieHeader}}}).Cookies()

	var tokenStr string
	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			tokenStr = cookie.Value
			break
		}
	}
	if tokenStr == "" {
		return "", errors.New("no access_token found in cookies")
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return auth.JwtKey, nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}

	username, ok := claims["username"].(string)
	if !ok {
		return "", errors.New("username not found")
	}

	return username, nil
}

func SetupSocketIO(router *gin.Engine) {
	server := socketio.NewServer(nil)
	server.OnEvent("/", "join", func(s socketio.Conn, _ string) {
		username, err := getUsernameFromSocket(s)
		if err != nil {
			s.Emit("error", "Unauthorized")
			return
		}

		s.Join("general")
		joinMsg := username + " a rejoint le salon."
		s.Emit("message", joinMsg)
		server.BroadcastToRoom("/", "general", "message", joinMsg)
	})

	server.OnEvent("/", "message", func(s socketio.Conn, msg string) {
		username, err := getUsernameFromSocket(s)
		if err != nil {
			s.Emit("error", "Unauthorized")
			return
		}

		fullMsg := username + ": " + msg
		server.BroadcastToRoom("/", "general", "message", fullMsg)
	})

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		return nil
	})

	server.OnEvent("/", "join", func(s socketio.Conn, username string) {
		s.Join("general")
		joinMsg := username + " a rejoint le salon."
		s.Emit("message", joinMsg)
		server.BroadcastToRoom("/", "general", "message", joinMsg)
	})

	server.OnEvent("/", "message", func(s socketio.Conn, data map[string]string) {
		username := data["username"]
		msg := data["message"]
		fullMsg := username + ": " + msg
		server.BroadcastToRoom("/", "general", "message", fullMsg)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("Disconnected: ", reason)
		s.Leave("general")
	})

	go func() {
		err := server.Serve()
		if err != nil {

		}
	}()

	router.GET("/socket.io/*any", gin.WrapH(server))
	router.POST("/socket.io/*any", gin.WrapH(server))
}
