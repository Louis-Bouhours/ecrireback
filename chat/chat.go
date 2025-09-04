package chat

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Louis-Bouhours/ecrireback/auth"
	"github.com/Louis-Bouhours/ecrireback/db"
	"github.com/Louis-Bouhours/ecrireback/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WSMessage struct {
	Username  string    `json:"username"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	Room      string    `json:"room"`
}

type WSUser struct {
	ID            string `json:"id,omitempty"`
	Username      string `json:"username"`
	Email         string `json:"email,omitempty"`
	Avatar        string `json:"avatar,omitempty"`
	Authenticated bool   `json:"authenticated"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type hub struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]WSUser
}

var wsHub = &hub{conns: make(map[*websocket.Conn]WSUser)}

func (h *hub) add(c *websocket.Conn, u WSUser) {
	h.mu.Lock()
	h.conns[c] = u
	h.mu.Unlock()
}
func (h *hub) remove(c *websocket.Conn) (u WSUser) {
	h.mu.Lock()
	u = h.conns[c]
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
func (h *hub) broadcastExcept(msg WSMessage, except *websocket.Conn) {
	h.mu.Lock()
	for c := range h.conns {
		if c == except {
			continue
		}
		if err := c.WriteJSON(msg); err != nil {
			log.Printf("WS write error: %v", err)
			_ = c.Close()
			delete(h.conns, c)
		}
	}
	h.mu.Unlock()
}

func maskToken(t string) string {
	if len(t) <= 12 {
		return t
	}
	return t[:6] + "..." + t[len(t)-6:]
}

func extractUserFromRequest(r *http.Request) WSUser {
	resolve := func(cl *auth.Claims, src string) WSUser {
		var user models.User
		found := false
		if cl.UserID != "" {
			if oid, err := primitive.ObjectIDFromHex(cl.UserID); err == nil {
				if err := db.UsersCol.FindOne(r.Context(), bson.M{"_id": oid}).Decode(&user); err == nil {
					found = true
					log.Printf("[WS Auth] %s: user found by _id", src)
				}
			}
		}
		if !found && cl.Username != "" {
			if err := db.UsersCol.FindOne(r.Context(), bson.M{"username": cl.Username}).Decode(&user); err == nil {
				log.Printf("[WS Auth] %s: user found by username", src)
			}
		}
		u := WSUser{
			ID:            cl.UserID,
			Username:      cl.Username,
			Authenticated: true,
		}
		if user.ID != primitive.NilObjectID {
			u.ID = user.ID.Hex()
		}
		u.Email = user.Email
		u.Avatar = user.Avatar
		return u
	}

	if ck := r.Header.Get("Cookie"); ck != "" {
		log.Printf("[WS Auth] Cookie header: %s", ck)
	}

	// 1) Cookie access_token
	if cookie, err := r.Cookie("access_token"); err == nil && cookie.Value != "" {
		log.Printf("[WS Auth] Found access_token cookie: %s", maskToken(cookie.Value))
		if claims, err := auth.ValidateJWT(cookie.Value); err == nil && claims.TokenType == "access" && claims.Username != "" {
			return resolve(claims, "Cookie")
		} else if err != nil {
			log.Printf("[WS Auth] access_token invalid: %v", err)
		}
	}

	// 2) Authorization: Bearer
	if authz := r.Header.Get("Authorization"); authz != "" {
		parts := strings.SplitN(authz, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			log.Printf("[WS Auth] Found Bearer: %s", maskToken(parts[1]))
			if claims, err := auth.ValidateJWT(parts[1]); err == nil && claims.TokenType == "access" && claims.Username != "" {
				return resolve(claims, "Authorization")
			}
		}
	}

	// 3) Query token
	if token := r.URL.Query().Get("token"); token != "" {
		log.Printf("[WS Auth] Found token query: %s", maskToken(token))
		if claims, err := auth.ValidateJWT(token); err == nil && claims.TokenType == "access" && claims.Username != "" {
			return resolve(claims, "Query")
		}
	}

	log.Printf("[WS Auth] guest")
	return WSUser{Username: "Invité", Authenticated: false}
}

func RegisterWS(router *gin.Engine) {
	router.GET("/ws", func(c *gin.Context) {
		log.Printf("[WS] Handshake from %s UA=%s", c.ClientIP(), c.Request.UserAgent())

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WS upgrade error: %v", err)
			return
		}
		log.Printf("WS connected: %s", c.ClientIP())

		user := extractUserFromRequest(c.Request)
		wsHub.add(conn, user)
		wsHub.broadcast(WSMessage{
			Username:  "Serveur",
			Text:      user.Username + " a rejoint le salon.",
			Timestamp: time.Now(),
			Room:      "general",
		})

		for {
			var in struct {
				Text string `json:"text"`
				Room string `json:"room"`
				// Optionnel si tu veux que le client envoie un username
				Username string `json:"username"`
			}
			if err := conn.ReadJSON(&in); err != nil {
				left := wsHub.remove(conn)
				_ = conn.Close()
				log.Printf("WS closed: %s (user=%s)", c.ClientIP(), left.Username)
				wsHub.broadcast(WSMessage{
					Username:  "Serveur",
					Text:      left.Username + " a quitté le salon.",
					Timestamp: time.Now(),
					Room:      "general",
				})
				return
			}

			room := in.Room
			if room == "" {
				room = "general"
			}
			// On privilégie l'identité authentifiée
			sender := user.Username
			if sender == "" || sender == "Invité" {
				if in.Username != "" {
					sender = in.Username
				} else {
					sender = "Invité"
				}
			}

			wsHub.broadcastExcept(WSMessage{
				Username:  sender,
				Text:      in.Text,
				Timestamp: time.Now(),
				Room:      room,
			}, conn)
		}
	})
}
