package sse

import (
	"fmt"
	"sync"
	"time"
)

// Event represents a Server-Sent Event
type Event struct {
	ID    string
	Event string
	Data  string
	Retry int // milliseconds
}

// Client represents an SSE client connection
type Client struct {
	ID        string
	Channel   chan *Event
	LastID    string
	closeCh   chan struct{}
	closeOnce sync.Once
}

// NewClient creates a new SSE client
func NewClient(id string, bufferSize int) *Client {
	if bufferSize <= 0 {
		bufferSize = 100
	}

	return &Client{
		ID:      id,
		Channel: make(chan *Event, bufferSize),
		closeCh: make(chan struct{}),
	}
}

// Close closes the client connection
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.closeCh)
		close(c.Channel)
	})
}

// IsClosed returns whether the client is closed
func (c *Client) IsClosed() bool {
	select {
	case <-c.closeCh:
		return true
	default:
		return false
	}
}

// Send sends an event to the client (non-blocking)
func (c *Client) Send(event *Event) bool {
	if c.IsClosed() {
		return false
	}

	select {
	case c.Channel <- event:
		return true
	default:
		return false
	}
}

// Broker manages SSE connections
type Broker struct {
	clients     sync.Map
	newClients  chan *Client
	deadClients chan *Client
	messages    chan *Event

	totalClients  int64
	messagesCount int64
	droppedCount  int64

	keepaliveInterval time.Duration
	maxClients        int
}

// NewBroker creates a new SSE broker
func NewBroker(maxClients int, keepaliveInterval time.Duration) *Broker {
	if maxClients <= 0 {
		maxClients = 10000
	}
	if keepaliveInterval <= 0 {
		keepaliveInterval = 30 * time.Second
	}

	broker := &Broker{
		newClients:        make(chan *Client, 100),
		deadClients:       make(chan *Client, 100),
		messages:          make(chan *Event, 1000),
		keepaliveInterval: keepaliveInterval,
		maxClients:        maxClients,
	}

	go broker.run()
	go broker.keepalive()

	return broker
}

func (b *Broker) run() {
	for {
		select {
		case client := <-b.newClients:
			b.clients.Store(client.ID, client)
			b.totalClients++

		case client := <-b.deadClients:
			b.clients.Delete(client.ID)
			client.Close()

		case event := <-b.messages:
			b.messagesCount++
			b.broadcast(event)
		}
	}
}

func (b *Broker) keepalive() {
	ticker := time.NewTicker(b.keepaliveInterval)
	defer ticker.Stop()

	for range ticker.C {
		keepaliveEvent := &Event{
			Event: "keepalive",
			Data:  fmt.Sprintf("timestamp:%d", time.Now().Unix()),
		}
		b.broadcast(keepaliveEvent)
	}
}

func (b *Broker) broadcast(event *Event) {
	b.clients.Range(func(key, value interface{}) bool {
		client := value.(*Client)
		if !client.Send(event) {
			b.droppedCount++
		}
		return true
	})
}

func (b *Broker) Register(client *Client) error {
	count := 0
	b.clients.Range(func(_, _ interface{}) bool {
		count++
		return true
	})

	if count >= b.maxClients {
		return fmt.Errorf("max clients reached (%d)", b.maxClients)
	}

	b.newClients <- client
	return nil
}

func (b *Broker) Unregister(client *Client) {
	b.deadClients <- client
}

func (b *Broker) Publish(event *Event) {
	b.messages <- event
}

func (b *Broker) PublishToClient(clientID string, event *Event) bool {
	val, ok := b.clients.Load(clientID)
	if !ok {
		return false
	}

	client := val.(*Client)
	return client.Send(event)
}

func (b *Broker) GetClient(clientID string) (*Client, bool) {
	val, ok := b.clients.Load(clientID)
	if !ok {
		return nil, false
	}
	return val.(*Client), true
}

func (b *Broker) ClientCount() int {
	count := 0
	b.clients.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

func (b *Broker) Stats() map[string]interface{} {
	return map[string]interface{}{
		"total_clients":    b.totalClients,
		"current_clients":  b.ClientCount(),
		"messages_sent":    b.messagesCount,
		"messages_dropped": b.droppedCount,
	}
}

func FormatEvent(event *Event) []byte {
	var buf []byte

	if event.ID != "" {
		buf = append(buf, []byte(fmt.Sprintf("id: %s\n", event.ID))...)
	}

	if event.Event != "" {
		buf = append(buf, []byte(fmt.Sprintf("event: %s\n", event.Event))...)
	}

	if event.Retry > 0 {
		buf = append(buf, []byte(fmt.Sprintf("retry: %d\n", event.Retry))...)
	}

	if event.Data != "" {
		buf = append(buf, []byte(fmt.Sprintf("data: %s\n", event.Data))...)
	}

	buf = append(buf, '\n')
	return buf
}
