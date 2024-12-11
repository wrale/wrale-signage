package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	v1alpha1 "github.com/wrale/wrale-signage/api/types/v1alpha1"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking
		return true
	},
}

// connection is an middleman between the websocket connection and the hub
type connection struct {
	displayID uuid.UUID
	ws        *websocket.Conn
	send      chan []byte
	hub       *Hub
	logger    *slog.Logger
}

func (c *connection) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.ws.Close()
	}()

	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				c.logger.Error("websocket read error",
					"error", err,
					"displayId", c.displayID,
				)
			}
			break
		}

		var status v1alpha1.ControlMessage
		if err := json.Unmarshal(message, &status); err != nil {
			c.logger.Error("invalid status message",
				"error", err,
				"displayId", c.displayID,
			)
			continue
		}

		if status.Type != v1alpha1.ControlMessageStatus {
			c.logger.Error("unexpected message type",
				"type", status.Type,
				"displayId", c.displayID,
			)
			continue
		}

		// Process display status update
		c.hub.broadcast <- message
	}
}

func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

func (c *connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// Hub maintains the set of active connections and broadcasts messages
type Hub struct {
	// Registered connections
	connections map[*connection]bool

	// Register requests from the connections
	register chan *connection

	// Unregister requests from connections
	unregister chan *connection

	// Inbound messages from the connections
	broadcast chan []byte

	// Logger instance
	logger *slog.Logger
}

func newHub(logger *slog.Logger) *Hub {
	return &Hub{
		broadcast:    make(chan []byte),
		register:     make(chan *connection),
		unregister:   make(chan *connection),
		connections:  make(map[*connection]bool),
		logger:       logger,
	}
}

func (h *Hub) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case c := <-h.register:
			h.connections[c] = true
			h.logger.Info("display connected",
				"displayId", c.displayID,
				"connections", len(h.connections),
			)
		case c := <-h.unregister:
			if _, ok := h.connections[c]; ok {
				delete(h.connections, c)
				close(c.send)
				h.logger.Info("display disconnected",
					"displayId", c.displayID,
					"connections", len(h.connections),
				)
			}
		case m := <-h.broadcast:
			for c := range h.connections {
				select {
				case c.send <- m:
				default:
					close(c.send)
					delete(h.connections, c)
				}
			}
		}
	}
}

// ServeWs handles websocket requests from displays
func (h *Handler) ServeWs(w http.ResponseWriter, r *http.Request) {
	displayID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "missing or invalid display ID", http.StatusBadRequest)
		return
	}

	// Verify display exists and is active
	display, err := h.service.Get(r.Context(), displayID)
	if err != nil {
		h.logger.Error("failed to get display",
			"error", err,
			"displayId", displayID,
		)
		http.Error(w, fmt.Sprintf("display not found: %s", displayID), http.StatusNotFound)
		return
	}

	if display.State != v1alpha1.DisplayStateActive {
		http.Error(w, "display not active", http.StatusForbidden)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed",
			"error", err,
			"displayId", displayID,
		)
		return
	}

	c := &connection{
		displayID: displayID,
		send:      make(chan []byte, 256),
		ws:        ws,
		hub:       h.hub,
		logger:    h.logger,
	}

	c.hub.register <- c

	go c.writePump()
	c.readPump()
}

// SendControlMessage sends a control message to a specific display
func (h *Handler) SendControlMessage(displayID uuid.UUID, message *v1alpha1.ControlMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal control message: %w", err)
	}

	// Find connection for display
	for c := range h.hub.connections {
		if c.displayID == displayID {
			select {
			case c.send <- data:
				return nil
			default:
				return fmt.Errorf("display connection buffer full: %s", displayID)
			}
		}
	}

	return fmt.Errorf("display not connected: %s", displayID)
}
