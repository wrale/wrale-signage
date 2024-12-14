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
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
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

	// WebSocket-specific close codes
	wsCloseRateLimitExceeded = 4429 // Custom code for rate limit
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking
		return true
	},
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

	// Rate limiting service
	rateLimit ratelimit.Service

	// Logger instance
	logger *slog.Logger
}

func newHub(rateLimit ratelimit.Service, logger *slog.Logger) *Hub {
	return &Hub{
		broadcast:   make(chan []byte),
		register:    make(chan *connection),
		unregister:  make(chan *connection),
		connections: make(map[*connection]bool),
		rateLimit:   rateLimit,
		logger:      logger,
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

// connection is an middleman between the websocket connection and the hub
type connection struct {
	displayID uuid.UUID
	ws        *websocket.Conn
	send      chan []byte
	hub       *Hub
	logger    *slog.Logger

	// Rate limit tracking
	readLimitKey  ratelimit.LimitKey
	writeLimitKey ratelimit.LimitKey
}

// cleanup handles proper connection closure and cleanup
func (c *connection) cleanup() {
	// Ensure we unregister before closing
	c.hub.unregister <- c

	// Close the websocket connection with proper error handling
	if err := c.ws.Close(); err != nil {
		c.logger.Error("error closing websocket connection",
			"error", err,
			"displayId", c.displayID,
		)
	}
}

func (c *connection) readPump() {
	defer c.cleanup()

	c.ws.SetReadLimit(maxMessageSize)
	if err := c.ws.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		c.logger.Error("failed to set read deadline",
			"error", err,
			"displayId", c.displayID,
		)
		return
	}

	c.ws.SetPongHandler(func(string) error {
		if err := c.ws.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			c.logger.Error("failed to set read deadline in pong handler",
				"error", err,
				"displayId", c.displayID,
			)
			return err
		}
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

		// Check rate limit before processing
		if err := c.hub.rateLimit.Allow(context.Background(), c.readLimitKey); err != nil {
			c.logger.Warn("read rate limit exceeded",
				"displayId", c.displayID,
				"error", err,
			)
			// Send rate limit error to client
			closeMsg := websocket.FormatCloseMessage(wsCloseRateLimitExceeded, "rate limit exceeded")
			if err := c.write(websocket.CloseMessage, closeMsg); err != nil {
				c.logger.Error("failed to send rate limit close message",
					"error", err,
					"displayId", c.displayID,
				)
			}
			return
		}

		// Process message
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
	if err := c.ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		c.logger.Error("failed to set write deadline",
			"error", err,
			"displayId", c.displayID,
		)
		return err
	}
	return c.ws.WriteMessage(mt, payload)
}

func (c *connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := c.ws.Close(); err != nil {
			c.logger.Error("error closing websocket connection in writePump",
				"error", err,
				"displayId", c.displayID,
			)
		}
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				if err := c.write(websocket.CloseMessage, []byte{}); err != nil {
					c.logger.Error("failed to write close message",
						"error", err,
						"displayId", c.displayID,
					)
				}
				return
			}

			// Check rate limit before sending
			if err := c.hub.rateLimit.Allow(context.Background(), c.writeLimitKey); err != nil {
				c.logger.Warn("write rate limit exceeded",
					"displayId", c.displayID,
					"error", err,
				)
				// Drop message and continue
				continue
			}

			if err := c.write(websocket.TextMessage, message); err != nil {
				c.logger.Error("failed to write message",
					"error", err,
					"displayId", c.displayID,
				)
				return
			}

		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				c.logger.Error("failed to write ping",
					"error", err,
					"displayId", c.displayID,
				)
				return
			}
		}
	}
}

// ServeWs handles websocket requests from displays
func (h *Handler) ServeWs(w http.ResponseWriter, r *http.Request) {
	// Get authenticated display ID from context (set by auth middleware)
	displayID, ok := GetDisplayID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Verify display exists and is active
	d, err := h.service.Get(r.Context(), displayID)
	if err != nil {
		h.logger.Error("failed to get display",
			"error", err,
			"displayId", displayID,
		)
		http.Error(w, fmt.Sprintf("display not found: %s", displayID), http.StatusNotFound)
		return
	}

	if convert(d.State) != v1alpha1.DisplayStateActive {
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

	// Create rate limit keys for this connection
	readKey := ratelimit.LimitKey{
		Type:  "ws_message_in",
		Token: displayID.String(),
	}
	writeKey := ratelimit.LimitKey{
		Type:  "ws_message_out",
		Token: displayID.String(),
	}

	c := &connection{
		displayID:     displayID,
		send:          make(chan []byte, 256),
		ws:            ws,
		hub:           h.hub,
		logger:        h.logger,
		readLimitKey:  readKey,
		writeLimitKey: writeKey,
	}

	h.hub.register <- c

	go c.writePump()
	c.readPump()
}

// convert converts between domain and API display states
func convert(s display.State) v1alpha1.DisplayState {
	switch s {
	case display.StateUnregistered:
		return v1alpha1.DisplayStateUnregistered
	case display.StateActive:
		return v1alpha1.DisplayStateActive
	case display.StateOffline:
		return v1alpha1.DisplayStateOffline
	case display.StateDisabled:
		return v1alpha1.DisplayStateDisabled
	default:
		return v1alpha1.DisplayStateOffline
	}
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
