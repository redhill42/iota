package devices

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redhill42/iota/device"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
)

type Hub struct {
	// Registered subscribers
	subscribers map[*Subscriber]bool

	// Register requests from the subscribers
	register chan *Subscriber

	// Unregister requests from subscribers
	unregister chan *Subscriber

	// Device attribute updates notification
	updates chan device.Record
}

type Subscriber struct {
	hub *Hub

	// The device id to subscribe, or nil for all devices
	id string

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (sub *Subscriber) readPump() {
	defer func() {
		sub.hub.unregister <- sub
		sub.conn.Close()
	}()

	sub.conn.SetReadLimit(maxMessageSize)
	sub.conn.SetReadDeadline(time.Now().Add(pongWait))
	sub.conn.SetPongHandler(func(string) error {
		sub.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := sub.conn.ReadMessage() // message discarded
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logrus.Error(err)
			}
			break
		}
	}
}

// writePump pumps messages from the Hub to the websocket connection
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine
func (sub *Subscriber) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		sub.conn.Close()
	}()

	for {
		select {
		case message, ok := <-sub.send:
			sub.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				sub.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := sub.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(sub.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-sub.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			sub.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := sub.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func newHub() *Hub {
	return &Hub{
		subscribers: make(map[*Subscriber]bool),
		register:    make(chan *Subscriber),
		unregister:  make(chan *Subscriber),
		updates:     make(chan device.Record),
	}
}

func (h *Hub) run() {
	for {
		select {
		case sub := <-h.register:
			h.subscribers[sub] = true
		case sub := <-h.unregister:
			if _, ok := h.subscribers[sub]; ok {
				delete(h.subscribers, sub)
				close(sub.send)
			}
		case updates := <-h.updates:
			message, err := json.Marshal(updates)
			if err != nil {
				logrus.Error(err)
				continue
			}

			id, _ := updates["id"].(string)
			for sub := range h.subscribers {
				if sub.id == "+" || sub.id == id {
					select {
					case sub.send <- message:
					default:
						close(sub.send)
						delete(h.subscribers, sub)
					}
				}
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *Hub) serveWs(w http.ResponseWriter, r *http.Request, id string) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	sub := &Subscriber{hub: h, id: id, conn: conn, send: make(chan []byte, 256)}
	sub.hub.register <- sub

	go sub.writePump()
	go sub.readPump()
	return nil
}
