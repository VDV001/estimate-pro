package ws

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/VDV001/estimate-pro/backend/pkg/jwt"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins in dev
	},
}

type Handler struct {
	hub        *Hub
	jwtService *jwt.Service
	getProjects func(userID string) []string // returns project IDs for user
}

func NewHandler(hub *Hub, jwtService *jwt.Service, getProjects func(userID string) []string) *Handler {
	return &Handler{hub: hub, jwtService: jwtService, getProjects: getProjects}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/api/v1/ws", h.ServeWS)
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Auth via query param: ?token=<jwt>
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	claims, err := h.jwtService.ValidateAccess(tokenStr)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade failed", "error", err)
		return
	}

	// Get user's projects
	projectIDs := make(map[string]bool)
	for _, pid := range h.getProjects(claims.UserID) {
		projectIDs[pid] = true
	}

	client := &Client{
		UserID:     claims.UserID,
		ProjectIDs: projectIDs,
		Send:       make(chan []byte, 256),
	}

	h.hub.Register(client)

	go h.writePump(conn, client)
	go h.readPump(conn, client)
}

func (h *Handler) writePump(conn *websocket.Conn, client *Client) {
	defer conn.Close()

	for msg := range client.Send {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (h *Handler) readPump(conn *websocket.Conn, client *Client) {
	defer func() {
		h.hub.Unregister(client)
		conn.Close()
	}()

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Keep reading to detect disconnect. We don't expect client messages.
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
