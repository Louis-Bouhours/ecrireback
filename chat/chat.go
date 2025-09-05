package chat

import (
	"context"
	"log"
	"net/http"
	"strconv"
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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

// ---------- Persistence async (non-bloquante) ----------

type persistItem struct {
	UserID    string
	Username  string
	Room      string
	Text      string
	Timestamp time.Time
}

var (
	persistQueue      = make(chan persistItem, 1000)
	persistWorkerOnce sync.Once
)

func startPersistenceWorker() {
	persistWorkerOnce.Do(func() {
		go func() {
			for it := range persistQueue {
				// Construit le document à insérer.
				// On conserve les champs du modèle existant (sender/content/created_at)
				// et on ajoute user_id/username/room pour requêtes futures.
				doc := bson.M{
					"sender":         it.Username,                                 // compat: nom de l'expéditeur
					"content":        it.Text,                                     // compat
					"created_at":     primitive.NewDateTimeFromTime(it.Timestamp), // compat
					"user_id":        it.UserID,                                   // nouvel attribut
					"username":       it.Username,                                 // redondant mais pratique
					"room":           it.Room,                                     // salon
					"created_at_iso": it.Timestamp,                                // lecture humaine si besoin
				}

				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				_, err := db.MessagesCol.InsertOne(ctx, doc)
				cancel()
				if err != nil {
					log.Printf("persist message failed: %v", err)
				}
			}
		}()
	})
}

// ------------------------------------------------------

func RegisterWS(router *gin.Engine) {
	// Démarre le worker de persistance une seule fois
	startPersistenceWorker()

	// Endpoint REST pour charger l'historique par room
	router.GET("/api/messages", func(c *gin.Context) {
		room := c.Query("room")
		if room == "" {
			room = "general"
		}
		limit := int64(100)
		if s := c.Query("limit"); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 500 {
				limit = int64(n)
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		opts := options.Find().
			SetSort(bson.D{{Key: "created_at", Value: 1}}). // ancien -> récent
			SetLimit(limit)

		cur, err := db.MessagesCol.Find(ctx, bson.M{"room": room}, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		defer func(cur *mongo.Cursor, ctx context.Context) {
			err := cur.Close(ctx)
			if err != nil {

			}
		}(cur, ctx)

		type msgDoc struct {
			ID        primitive.ObjectID `bson:"_id"`
			Sender    string             `bson:"sender"`
			Content   string             `bson:"content"`
			Room      string             `bson:"room"`
			CreatedAt primitive.DateTime `bson:"created_at"`
		}

		out := make([]gin.H, 0, limit)
		for cur.Next(ctx) {
			var doc msgDoc
			if err := cur.Decode(&doc); err != nil {
				continue
			}
			out = append(out, gin.H{
				"id":        doc.ID.Hex(),
				"username":  doc.Sender,
				"text":      doc.Content,
				"timestamp": doc.CreatedAt.Time().UTC().Format(time.RFC3339),
				"room":      doc.Room,
			})
		}
		if err := cur.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db cursor error"})
			return
		}

		c.JSON(http.StatusOK, out)
	})

	// WebSocket temps réel
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
				Text     string `json:"text"`
				Room     string `json:"room"`
				Username string `json:"username"` // facultatif, on privilégie l'identité auth
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
			// Identité: priorité à l'utilisateur authentifié
			sender := user.Username
			if sender == "" || sender == "Invité" {
				if in.Username != "" {
					sender = in.Username
				} else {
					sender = "Invité"
				}
			}

			ts := time.Now().UTC()

			// Diffuse en temps réel (sauf à l'émetteur, qui gère un écho local côté client)
			wsHub.broadcastExcept(WSMessage{
				Username:  sender,
				Text:      in.Text,
				Timestamp: ts,
				Room:      room,
			}, conn)

			// Persistance asynchrone: ne bloque pas le flux WS.
			select {
			case persistQueue <- persistItem{
				UserID:    user.ID, // vide si invité
				Username:  sender,  // username affiché
				Room:      room,
				Text:      in.Text,
				Timestamp: ts,
			}:
			default:
				// File pleine: on drop et on log (stratégie simple, à ajuster si besoin)
				log.Printf("persist queue full: dropping message from user=%s", sender)
			}
		}
	})
}
