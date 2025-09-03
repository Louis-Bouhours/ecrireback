package chat

import (
	"log"
	"net/http"
	"time"

	"github.com/Louis-Bouhours/ecrireback/auth" // Package pour la validation JWT
	"github.com/Louis-Bouhours/ecrireback/db"   // Package pour l'accès à la BDD
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChatMessage définit la structure d'un message sauvegardé en base de données.
type ChatMessage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Username  string             `bson:"username"`
	Text      string             `bson:"text"`
	Timestamp time.Time          `bson:"timestamp"`
}

// UserContext stocke les informations de l'utilisateur authentifié pour une connexion socket.
type UserContext struct {
	UserID   string
	Username string
}

// SetupSocketIO configure le serveur Socket.IO, ses événements et l'attache au routeur Gin.
func SetupSocketIO(router *gin.Engine) {
	server := socketio.NewServer(nil)

	// --- ÉVÉNEMENT PRINCIPAL : CONNEXION ---
	// C'est ici que l'on gère l'authentification. Si elle échoue, la connexion est refusée.
	server.OnConnect("/", func(s socketio.Conn) error {
		// 1. Extraire le cookie de la requête initiale
		req := s.RemoteHeader()
		cookieHeader := req.Get("Cookie")
		cookie, err := (&http.Request{Header: http.Header{"Cookie": {cookieHeader}}}).Cookie("token")
		if err != nil {
			log.Println("Socket connection refused: no token cookie found")
			return err // Refuse la connexion
		}

		// 2. Valider le token JWT
		claims, err := auth.ValidateJWT(cookie.Value)
		if err != nil {
			log.Println("Socket connection refused: invalid token")
			return err // Refuse la connexion
		}

		// 3. Stocker les informations de l'utilisateur dans le contexte du socket
		// C'est sécurisé car validé côté serveur.
		ctx := &UserContext{
			UserID:   claims.UserID,
			Username: claims.Username,
		}
		s.SetContext(ctx)

		// 4. L'utilisateur rejoint le salon de discussion principal
		s.Join("general")
		log.Printf("Socket connected: %s (ID: %s)", ctx.Username, s.ID())

		// 5. Envoyer l'historique des messages à l'utilisateur qui vient de se connecter
		history, err := loadMessageHistory()
		if err != nil {
			log.Printf("Error loading message history: %v", err)
		} else {
			s.Emit("history", history)
		}

		// 6. Annoncer à tout le monde que l'utilisateur a rejoint
		joinMsg := ctx.Username + " a rejoint le salon."
		server.BroadcastToRoom("/", "general", "chat message", ChatMessage{
			Username: "Serveur", Text: joinMsg, Timestamp: time.Now(),
		})

		return nil
	})

	// --- ÉVÉNEMENT : ENVOI D'UN MESSAGE ---
	server.OnEvent("/", "chat message", func(s socketio.Conn, msg string) {
		// On récupère le contexte de l'utilisateur (défini à la connexion)
		ctx, ok := s.Context().(*UserContext)
		if !ok || ctx == nil {
			s.Emit("error", "Unauthorized")
			return
		}

		// On crée l'objet message complet
		chatMsg := ChatMessage{
			Username:  ctx.Username,
			Text:      msg,
			Timestamp: time.Now(),
		}

		// On le sauvegarde en base de données
		go saveMessage(chatMsg) // go routine pour ne pas bloquer

		// On le diffuse à tous les utilisateurs dans le salon
		server.BroadcastToRoom("/", "general", "chat message", chatMsg)
	})

	// --- ÉVÉNEMENT : DÉCONNEXION ---
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		ctx, ok := s.Context().(*UserContext)
		if ok && ctx != nil {
			log.Printf("Socket disconnected: %s (Reason: %s)", ctx.Username, reason)
			leaveMsg := ctx.Username + " a quitté le salon."
			server.BroadcastToRoom("/", "general", "chat message", ChatMessage{
				Username: "Serveur", Text: leaveMsg, Timestamp: time.Now(),
			})
		}
		s.Leave("general")
	})

	// --- GESTION DES ERREURS ---
	server.OnError("/", func(s socketio.Conn, e error) {
		log.Printf("Socket error: %v", e)
	})

	// Démarrage du serveur Socket.IO dans une goroutine
	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("Socket.IO listen error: %s\n", err)
		}
	}()

	// Attachement du serveur Socket.IO au routeur Gin
	router.GET("/socket.io/*any", gin.WrapH(server))
	router.POST("/socket.io/*any", gin.WrapH(server))
}

// saveMessage insère un message dans la collection MongoDB.
func saveMessage(msg ChatMessage) {
	_, err := db.MessagesCol.InsertOne(db.Ctx, msg)
	if err != nil {
		log.Printf("Failed to save message to DB: %v", err)
	}
}

// loadMessageHistory récupère les 50 derniers messages de la base de données.
func loadMessageHistory() ([]ChatMessage, error) {
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(50)
	cursor, err := db.MessagesCol.Find(db.Ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(db.Ctx)

	var messages []ChatMessage
	if err = cursor.All(db.Ctx, &messages); err != nil {
		return nil, err
	}

	// Les messages sont du plus récent au plus ancien, on les inverse pour l'affichage
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
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
