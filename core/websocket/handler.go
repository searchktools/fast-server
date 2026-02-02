package websocket

import (
	"bufio"
	"net"
)

type Handler struct {
	hub *Hub
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub: hub,
	}
}

func (h *Handler) HandleConnection(conn net.Conn, clientID string) error {
	reader := bufio.NewReader(conn)
	wsConn, err := Upgrade(conn, reader)
	if err != nil {
		conn.Close()
		return err
	}

	client := NewClient(clientID, wsConn)

	if err := h.hub.Register(client); err != nil {
		wsConn.Close()
		return err
	}

	return nil
}

type MessageHandler func(client *Client, msg *Message)

type CustomHub struct {
	*Hub
	onMessage MessageHandler
}

func NewCustomHub(maxClients int, onMessage MessageHandler) *CustomHub {
	hub := NewHub(maxClients)

	customHub := &CustomHub{
		Hub:       hub,
		onMessage: onMessage,
	}

	return customHub
}

func (h *CustomHub) Register(client *Client) error {
	count := 0
	h.clients.Range(func(_, _ interface{}) bool {
		count++
		return true
	})

	if count >= h.maxClients {
		return nil
	}

	h.register <- client

	go h.customReadPump(client)
	go h.writePump(client)

	return nil
}

func (h *CustomHub) customReadPump(client *Client) {
	defer func() {
		h.Unregister(client)
	}()

	for {
		msg, err := client.Conn.ReadMessage()
		if err != nil {
			return
		}

		if h.onMessage != nil {
			h.onMessage(client, msg)
		}
	}
}

type EventType string

const (
	EventConnect    EventType = "connect"
	EventDisconnect EventType = "disconnect"
	EventMessage    EventType = "message"
	EventJoinRoom   EventType = "join"
	EventLeaveRoom  EventType = "leave"
	EventError      EventType = "error"
)

type Event struct {
	Type EventType              `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}
