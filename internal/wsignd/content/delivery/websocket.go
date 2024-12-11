package delivery

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log/slog"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

// Manager handles WebSocket connections for content delivery
type Manager struct {
	displayID uuid.UUID
	conn      *websocket.Conn
	sequence  chan *v1alpha1.ContentSequence
	errors    chan error
	done      chan struct{}
	logger    *slog.Logger
}

// NewManager creates a new content delivery manager
func NewManager(displayID uuid.UUID, logger *slog.Logger) *Manager {
	return &Manager{
		displayID: displayID,
		sequence:  make(chan *v1alpha1.ContentSequence, 1),
		errors:    make(chan error, 1),
		done:      make(chan struct{}),
		logger:    logger,
	}
}

// Connect establishes connection to control WebSocket
func (m *Manager) Connect(ctx context.Context, wsURL string) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	m.conn = conn

	// Start handling messages
	go m.readMessages()
	go m.writeStatus()

	return nil
}

// Close terminates the WebSocket connection
func (m *Manager) Close() error {
	close(m.done)
	if m.conn != nil {
		if err := m.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			m.logger.Error("error sending close message",
				"error", err,
				"displayId", m.displayID,
			)
		}
		if err := m.conn.Close(); err != nil {
			m.logger.Error("error closing websocket connection",
				"error", err,
				"displayId", m.displayID,
			)
			return err
		}
	}
	return nil
}

// GetSequence returns channel for receiving content sequences
func (m *Manager) GetSequence() <-chan *v1alpha1.ContentSequence {
	return m.sequence
}

// GetErrors returns channel for receiving connection errors
func (m *Manager) GetErrors() <-chan error {
	return m.errors
}

func (m *Manager) readMessages() {
	defer func() {
		if err := m.conn.Close(); err != nil {
			m.logger.Error("error closing websocket read connection",
				"error", err,
				"displayId", m.displayID,
			)
		}
	}()

	m.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	m.conn.SetPongHandler(func(string) error {
		return m.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	for {
		select {
		case <-m.done:
			return
		default:
			var msg v1alpha1.ControlMessage
			err := m.conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					m.logger.Error("websocket read error",
						"error", err,
						"displayId", m.displayID,
					)
				}
				m.errors <- err
				return
			}

			switch msg.Type {
			case v1alpha1.ControlMessageSequenceUpdate:
				if msg.Sequence != nil {
					m.sequence <- msg.Sequence
				}
			case v1alpha1.ControlMessageReload:
				// Signal page reload needed
				m.errors <- &ReloadRequiredError{At: time.Now()}
			}
		}
	}
}

func (m *Manager) writeStatus() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		if err := m.conn.Close(); err != nil {
			m.logger.Error("error closing websocket write connection",
				"error", err,
				"displayId", m.displayID,
			)
		}
	}()

	// Send initial status message
	if err := m.sendStatus("", nil); err != nil {
		m.logger.Error("error sending initial status",
			"error", err,
			"displayId", m.displayID,
		)
		m.errors <- err
		return
	}

	// Set up ping handler
	pingTicker := time.NewTicker(54 * time.Second) // 90% of read timeout
	defer pingTicker.Stop()

	for {
		select {
		case <-m.done:
			return
		case <-ticker.C:
			if err := m.sendStatus("", nil); err != nil {
				m.logger.Error("error sending status update",
					"error", err,
					"displayId", m.displayID,
				)
				m.errors <- err
				return
			}
		case <-pingTicker.C:
			if err := m.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				m.logger.Error("error sending ping",
					"error", err,
					"displayId", m.displayID,
				)
				m.errors <- err
				return
			}
		}
	}
}

func (m *Manager) sendStatus(currentURL string, lastErr *string) error {
	msg := v1alpha1.ControlMessage{
		Type: v1alpha1.ControlMessageStatus,
		TypeMeta: v1alpha1.TypeMeta{
			Kind:       "ControlMessage",
			APIVersion: "v1alpha1",
		},
		Timestamp: time.Now(),
		Status: &v1alpha1.ControlStatus{
			CurrentURL: currentURL,
			LastError:  lastErr,
			UpdatedAt:  time.Now(),
		},
	}

	if err := m.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		m.logger.Error("error setting write deadline",
			"error", err,
			"displayId", m.displayID,
		)
		return err
	}

	return m.conn.WriteJSON(msg)
}

// ReloadRequiredError indicates display should reload device URL
type ReloadRequiredError struct {
	At time.Time
}

func (e *ReloadRequiredError) Error() string {
	return "display reload required"
}