package chat

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Louis-Bouhours/ecrireback/auth"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Username  string    `json:"username"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	Room      string    `json:"room"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Autorise toutes les origines (dev)
		return true
	},
}

type hub struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]string
}

var wsHub = &hub{conns: make(map[*websocket.Conn]string)}

func (h *hub) add(c *websocket.Conn, username string) {
	h.mu.Lock()
	h.conns[c] = username
	h.mu.Unlock()
}

func (h *hub) remove(c *websocket.Conn) (username string) {
	h.mu.Lock()
	username = h.conns[c]
	delete(h.conns, c)
	h.mu.Unlock()
	return
}

func (h *hub) broadcast(msg WSMessage) {
	h.mu.Lock()
	for c := range h.conns {
		if err := c.WriteJSON(msg); err != nil {
			log.Printf("WS write error: %v", err)
			_ = c.Close()
			delete(h.conns, c)
		}
	}
	h.mu.Unlock()
}

func RegisterWS(router *gin.Engine) {
	router.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WS upgrade error (remote=%s, ua=%s): %v", c.ClientIP(), c.Request.UserAgent(), err)
			return
		}
		log.Printf("WS connected: %s", c.ClientIP())

		// Important: ne pas defer conn.Close() ici, on gère la fermeture dans la boucle
		username := "Invité"
		if cookie, err := c.Request.Cookie("access_token"); err == nil {
			if claims, err := auth.ValidateJWT(cookie.Value); err == nil && claims.Username != "" {
				username = claims.Username
			}
		}

		wsHub.add(conn, username)
		wsHub.broadcast(WSMessage{
			Username:  "Serveur",
			Text:      username + " a rejoint le salon.",
			Timestamp: time.Now(),
			Room:      "general",
		})

		// Reader loop
		for {
			var in struct {
				Text string `json:"text"`
				Room string `json:"room"`
			}
			if err := conn.ReadJSON(&in); err != nil {
				left := wsHub.remove(conn)
				_ = conn.Close()
				log.Printf("WS closed: %s (%s)", c.ClientIP(), left)
				wsHub.broadcast(WSMessage{
					Username:  "Serveur",
					Text:      left + " a quitté le salon.",
					Timestamp: time.Now(),
					Room:      "general",
				})
				return
			}
			room := in.Room
			if room == "" {
				room = "general"
			}
			wsHub.broadcast(WSMessage{
				Username:  username,
				Text:      in.Text,
				Timestamp: time.Now(),
				Room:      room,
			})
		}
	})
}
