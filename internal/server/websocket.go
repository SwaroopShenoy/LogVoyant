package server

import (
	"log"
	"net/http"
	"net/url"
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
	clients    map[string]map[*websocket.Conn]bool
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
	
	decodedStreamID, err := url.QueryUnescape(streamID)
	if err != nil {
		log.Printf("Failed to decode stream ID: %v", err)
		decodedStreamID = streamID
	}
	
	log.Printf("WebSocket connection for stream: %s (decoded: %s)", streamID, decodedStreamID)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{conn: conn, streamID: decodedStreamID}
	s.hub.register <- client

	logs, err := s.config.Storage.GetLogs(decodedStreamID, storage.GetLogsOptions{Limit: 100})
	log.Printf("Attempting to fetch logs for stream: %s, found: %d, err: %v", decodedStreamID, len(logs), err)
	
	if err == nil && len(logs) > 0 {
		log.Printf("Sending %d historical logs to WebSocket client", len(logs))
		for _, logLine := range logs {
			if err := conn.WriteJSON(logLine); err != nil {
				log.Printf("Failed to send log: %v", err)
				break
			}
		}
	} else {
		log.Printf("No logs found for stream: %s (err: %v)", decodedStreamID, err)
	}

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			s.hub.unregister <- client
			break
		}
	}
}