package sse

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Stream struct {
	broker    *Broker
	eventID   atomic.Uint64
	namespace string
}

func NewStream(namespace string) *Stream {
	return &Stream{
		broker:    NewBroker(10000, 30*time.Second),
		namespace: namespace,
	}
}

func (s *Stream) WithBroker(broker *Broker) *Stream {
	s.broker = broker
	return s
}

func (s *Stream) Subscribe(clientID string) (*Client, error) {
	client := NewClient(clientID, 100)
	err := s.broker.Register(client)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (s *Stream) Unsubscribe(client *Client) {
	s.broker.Unregister(client)
}

func (s *Stream) Send(eventType, data string) error {
	id := s.eventID.Add(1)
	event := &Event{
		ID:    fmt.Sprintf("%s-%d", s.namespace, id),
		Event: eventType,
		Data:  data,
	}

	s.broker.Publish(event)
	return nil
}

func (s *Stream) SendTo(clientID, eventType, data string) error {
	id := s.eventID.Add(1)
	event := &Event{
		ID:    fmt.Sprintf("%s-%d", s.namespace, id),
		Event: eventType,
		Data:  data,
	}

	if !s.broker.PublishToClient(clientID, event) {
		return fmt.Errorf("client not found or channel full")
	}
	return nil
}

func (s *Stream) Broadcast(message string) error {
	return s.Send("message", message)
}

func (s *Stream) ClientCount() int {
	return s.broker.ClientCount()
}

func (s *Stream) Stats() map[string]interface{} {
	stats := s.broker.Stats()
	stats["namespace"] = s.namespace
	stats["event_id"] = s.eventID.Load()
	return stats
}

type Room struct {
	name    string
	clients sync.Map
	stream  *Stream
}

func NewRoom(name string, stream *Stream) *Room {
	return &Room{
		name:   name,
		stream: stream,
	}
}

func (r *Room) Join(client *Client) {
	r.clients.Store(client.ID, client)
}

func (r *Room) Leave(clientID string) {
	r.clients.Delete(clientID)
}

func (r *Room) Broadcast(eventType, data string) {
	event := &Event{
		Event: eventType,
		Data:  data,
	}

	r.clients.Range(func(key, value interface{}) bool {
		client := value.(*Client)
		client.Send(event)
		return true
	})
}

func (r *Room) ClientCount() int {
	count := 0
	r.clients.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}
