package server

import (
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"logvoyant/internal/storage"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type WebSocketHub struct {
	clients    map[string]map[*websocket.Conn]bool // streamID -> connections
	broadcast  chan LogBroadcast
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

type Client struct {
	conn     *websocket.Conn
	streamID string
}

type LogBroadcast struct {
	StreamID string
	Log      storage.LogLine
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[string]map[*websocket.Conn]bool),
		broadcast:  make(chan LogBroadcast, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.streamID] == nil {
				h.clients[client.streamID] = make(map[*websocket.Conn]bool)
			}
			h.clients[client.streamID][client.conn] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.streamID]; ok {
				if _, ok := clients[client.conn]; ok {
					delete(clients, client.conn)
					client.conn.Close()
					if len(clients) == 0 {
						delete(h.clients, client.streamID)
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[msg.StreamID]
			h.mu.RUnlock()

			for conn := range clients {
				err := conn.WriteJSON(msg.Log)
				if err != nil {
					log.Printf("WebSocket write error: %v", err)
					h.unregister <- &Client{conn: conn, streamID: msg.StreamID}
				}
			}
		}
	}
}

func (h *WebSocketHub) BroadcastLog(streamID string, log storage.LogLine) {
	h.broadcast <- LogBroadcast{
		StreamID: streamID,
		Log:      log,
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{conn: conn, streamID: streamID}
	s.hub.register <- client

	// Send recent logs on connect
	logs, err := s.config.Storage.GetLogs(streamID, storage.GetLogsOptions{Limit: 100})
	if err == nil {
		for _, log := range logs {
			conn.WriteJSON(log)
		}
	}

	// Keep connection alive
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			s.hub.unregister <- client
			break
		}
	}
}