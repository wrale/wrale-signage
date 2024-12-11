package delivery

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

// Manager handles WebSocket connections for content delivery
type Manager struct {
	displayID uuid.UUID
	conn      *websocket.Conn
	sequence  chan *v1alpha1.ContentSequence
	errors    chan error
	done      chan struct{}
}

// NewManager creates a new content delivery manager
func NewManager(displayID uuid.UUID) *Manager {
	return &Manager{
		displayID: displayID,
		sequence:  make(chan *v1alpha1.ContentSequence, 1),
		errors:    make(chan error, 1),
		done:      make(chan struct{}),
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
		return m.conn.Close()
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
	defer m.conn.Close()

	for {
		select {
		case <-m.done:
			return
		default:
			var msg v1alpha1.ControlMessage
			err := m.conn.ReadJSON(&msg)
			if err != nil {
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
	defer m.conn.Close()

	// Initial status message
	if err := m.sendStatus("", nil); err != nil {
		m.errors <- err
		return
	}

	for {
		select {
		case <-m.done:
			return
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

	return m.conn.WriteJSON(msg)
}

// ReloadRequiredError indicates display should reload device URL
type ReloadRequiredError struct {
	At time.Time
}

func (e *ReloadRequiredError) Error() string {
	return "display reload required"
}
