package websocket

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type Client struct {
	ID     string
	Conn   *Conn
	Send   chan []byte
	closed atomic.Bool
}

func NewClient(id string, conn *Conn) *Client {
	return &Client{
		ID:   id,
		Conn: conn,
		Send: make(chan []byte, 256),
	}
}

func (c *Client) Close() {
	if c.closed.Swap(true) {
		return
	}
	close(c.Send)
	c.Conn.Close()
}

func (c *Client) IsClosed() bool {
	return c.closed.Load()
}

type Hub struct {
	clients    sync.Map
	broadcast  chan *BroadcastMessage
	register   chan *Client
	unregister chan *Client
	rooms      sync.Map

	totalClients atomic.Int64
	messageCount atomic.Int64
	maxClients   int
}

type BroadcastMessage struct {
	OpCode  OpCode
	Payload []byte
	Room    string
}

func NewHub(maxClients int) *Hub {
	if maxClients <= 0 {
		maxClients = 10000
	}

	hub := &Hub{
		broadcast:  make(chan *BroadcastMessage, 1000),
		register:   make(chan *Client, 100),
		unregister: make(chan *Client, 100),
		maxClients: maxClients,
	}

	go hub.run()

	return hub
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients.Store(client.ID, client)
			h.totalClients.Add(1)

		case client := <-h.unregister:
			if _, ok := h.clients.Load(client.ID); ok {
				h.clients.Delete(client.ID)
				client.Close()
			}

		case msg := <-h.broadcast:
			h.messageCount.Add(1)

			if msg.Room == "" {
				h.clients.Range(func(key, value interface{}) bool {
					client := value.(*Client)
					select {
					case client.Send <- msg.Payload:
					default:
						h.unregister <- client
					}
					return true
				})
			} else {
				if room, ok := h.GetRoom(msg.Room); ok {
					room.Broadcast(msg.Payload)
				}
			}
		}
	}
}

func (h *Hub) Register(client *Client) error {
	count := 0
	h.clients.Range(func(_, _ interface{}) bool {
		count++
		return true
	})

	if count >= h.maxClients {
		return fmt.Errorf("max clients reached (%d)", h.maxClients)
	}

	h.register <- client

	go h.readPump(client)
	go h.writePump(client)

	return nil
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Broadcast(opcode OpCode, payload []byte, room string) {
	h.broadcast <- &BroadcastMessage{
		OpCode:  opcode,
		Payload: payload,
		Room:    room,
	}
}

func (h *Hub) BroadcastText(text string, room string) {
	h.Broadcast(OpText, []byte(text), room)
}

func (h *Hub) BroadcastBinary(data []byte, room string) {
	h.Broadcast(OpBinary, data, room)
}

func (h *Hub) SendTo(clientID string, payload []byte) error {
	val, ok := h.clients.Load(clientID)
	if !ok {
		return fmt.Errorf("client not found: %s", clientID)
	}

	client := val.(*Client)

	select {
	case client.Send <- payload:
		return nil
	default:
		return fmt.Errorf("client channel full")
	}
}

func (h *Hub) GetClient(clientID string) (*Client, bool) {
	val, ok := h.clients.Load(clientID)
	if !ok {
		return nil, false
	}
	return val.(*Client), true
}

func (h *Hub) ClientCount() int {
	count := 0
	h.clients.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

func (h *Hub) Stats() map[string]interface{} {
	return map[string]interface{}{
		"total_clients":   h.totalClients.Load(),
		"current_clients": h.ClientCount(),
		"messages_sent":   h.messageCount.Load(),
		"rooms":           h.RoomCount(),
	}
}

func (h *Hub) readPump(client *Client) {
	defer func() {
		h.Unregister(client)
	}()

	for {
		msg, err := client.Conn.ReadMessage()
		if err != nil {
			return
		}
		_ = msg
	}
}

func (h *Hub) writePump(client *Client) {
	defer func() {
		h.Unregister(client)
	}()

	for payload := range client.Send {
		if err := client.Conn.WriteMessage(OpText, payload); err != nil {
			return
		}
	}
}

type Room struct {
	Name    string
	clients sync.Map
	hub     *Hub
}

func (h *Hub) CreateRoom(name string) *Room {
	room := &Room{
		Name: name,
		hub:  h,
	}
	h.rooms.Store(name, room)
	return room
}

func (h *Hub) GetRoom(name string) (*Room, bool) {
	val, ok := h.rooms.Load(name)
	if !ok {
		return nil, false
	}
	return val.(*Room), true
}

func (h *Hub) DeleteRoom(name string) {
	if room, ok := h.GetRoom(name); ok {
		room.clients.Range(func(key, value interface{}) bool {
			room.Leave(key.(string))
			return true
		})
		h.rooms.Delete(name)
	}
}

func (h *Hub) RoomCount() int {
	count := 0
	h.rooms.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

func (r *Room) Join(clientID string) error {
	client, ok := r.hub.GetClient(clientID)
	if !ok {
		return fmt.Errorf("client not found: %s", clientID)
	}

	r.clients.Store(clientID, client)
	return nil
}

func (r *Room) Leave(clientID string) {
	r.clients.Delete(clientID)
}

func (r *Room) Broadcast(payload []byte) {
	r.clients.Range(func(key, value interface{}) bool {
		client := value.(*Client)
		select {
		case client.Send <- payload:
		default:
		}
		return true
	})
}

func (r *Room) BroadcastText(text string) {
	r.Broadcast([]byte(text))
}

func (r *Room) ClientCount() int {
	count := 0
	r.clients.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

func (r *Room) ClientIDs() []string {
	ids := make([]string, 0)
	r.clients.Range(func(key, value interface{}) bool {
		ids = append(ids, key.(string))
		return true
	})
	return ids
}
