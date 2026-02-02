package sse

import (
	"fmt"
	"time"
)

type Handler struct {
	stream *Stream
}

func NewHandler(stream *Stream) *Handler {
	return &Handler{
		stream: stream,
	}
}

func (h *Handler) HandleConnection(clientID string, onEvent func([]byte) error, onClose func()) error {
	client, err := h.stream.Subscribe(clientID)
	if err != nil {
		return err
	}
	defer func() {
		h.stream.Unsubscribe(client)
		if onClose != nil {
			onClose()
		}
	}()

	connectEvent := &Event{
		Event: "connected",
		Data:  fmt.Sprintf("client_id:%s", clientID),
	}

	if err := onEvent(FormatEvent(connectEvent)); err != nil {
		return err
	}

	for {
		select {
		case event, ok := <-client.Channel:
			if !ok {
				return nil
			}

			if err := onEvent(FormatEvent(event)); err != nil {
				return err
			}

		case <-client.closeCh:
			return nil
		}
	}
}

func WriteSSEHeaders() map[string]string {
	return map[string]string{
		"Content-Type":      "text/event-stream",
		"Cache-Control":     "no-cache",
		"Connection":        "keep-alive",
		"X-Accel-Buffering": "no",
	}
}

type EventBuilder struct {
	event *Event
}

func NewEventBuilder() *EventBuilder {
	return &EventBuilder{
		event: &Event{},
	}
}

func (eb *EventBuilder) WithID(id string) *EventBuilder {
	eb.event.ID = id
	return eb
}

func (eb *EventBuilder) WithEvent(eventType string) *EventBuilder {
	eb.event.Event = eventType
	return eb
}

func (eb *EventBuilder) WithData(data string) *EventBuilder {
	eb.event.Data = data
	return eb
}

func (eb *EventBuilder) WithRetry(ms int) *EventBuilder {
	eb.event.Retry = ms
	return eb
}

func (eb *EventBuilder) Build() *Event {
	return eb.event
}

func (eb *EventBuilder) Format() []byte {
	return FormatEvent(eb.event)
}

func NewMessageEvent(message string) *Event {
	return &Event{
		Event: "message",
		Data:  message,
	}
}

func NewNotificationEvent(title, body string) *Event {
	return &Event{
		Event: "notification",
		Data:  fmt.Sprintf(`{"title":"%s","body":"%s"}`, title, body),
	}
}

func NewHeartbeatEvent() *Event {
	return &Event{
		Event: "heartbeat",
		Data:  fmt.Sprintf("timestamp:%d", time.Now().Unix()),
	}
}

func NewErrorEvent(code int, message string) *Event {
	return &Event{
		Event: "error",
		Data:  fmt.Sprintf(`{"code":%d,"message":"%s"}`, code, message),
	}
}

func NewProgressEvent(current, total int, message string) *Event {
	return &Event{
		Event: "progress",
		Data:  fmt.Sprintf(`{"current":%d,"total":%d,"message":"%s"}`, current, total, message),
	}
}
