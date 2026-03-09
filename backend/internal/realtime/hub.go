package realtime

import (
	"encoding/json"
	"sync"

	"github.com/hertz-contrib/websocket"
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*websocket.Conn]*client
}

type client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*websocket.Conn]*client),
	}
}

func (h *Hub) Join(room string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*websocket.Conn]*client)
	}
	if _, ok := h.rooms[room][conn]; ok {
		return
	}
	h.rooms[room][conn] = &client{conn: conn}
}

func (h *Hub) Leave(room string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns := h.rooms[room]
	if conns == nil {
		return
	}
	delete(conns, conn)
	if len(conns) == 0 {
		delete(h.rooms, room)
	}
}

func (h *Hub) Broadcast(room string, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}

	h.mu.RLock()
	conns := h.rooms[room]
	var targets []*client
	for _, c := range conns {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		c.mu.Lock()
		err := c.conn.WriteMessage(websocket.TextMessage, raw)
		c.mu.Unlock()
		if err != nil {
			_ = c.conn.Close()
			h.Leave(room, c.conn)
		}
	}
}
