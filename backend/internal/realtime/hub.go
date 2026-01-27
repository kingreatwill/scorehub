package realtime

import (
	"encoding/json"
	"sync"

	"github.com/hertz-contrib/websocket"
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*websocket.Conn]struct{}),
	}
}

func (h *Hub) Join(room string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*websocket.Conn]struct{})
	}
	h.rooms[room][conn] = struct{}{}
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
	var targets []*websocket.Conn
	for c := range conns {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		if err := c.WriteMessage(websocket.TextMessage, raw); err != nil {
			_ = c.Close()
			h.Leave(room, c)
		}
	}
}

