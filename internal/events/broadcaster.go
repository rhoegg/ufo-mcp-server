package events

import (
	"encoding/json"
	"sync"
	"time"
)

// Event represents a state change event
type Event struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// EventType constants
const (
	EventEffectStarted   = "effect_started"
	EventEffectStopped   = "effect_stopped"
	EventEffectCompleted = "effect_completed"
	EventDimChanged      = "dim_changed"
	EventRingUpdate    = "ring_update"
	EventButtonPress   = "button_press"
	EventRawExecuted   = "raw_executed"
	EventProgress      = "progress"
)

// Subscriber represents a client listening for events
type Subscriber struct {
	ID      string
	Channel chan Event
}

// Broadcaster manages event distribution to multiple clients
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[string]*Subscriber
	eventChan   chan Event
}

// NewBroadcaster creates a new event broadcaster
func NewBroadcaster() *Broadcaster {
	b := &Broadcaster{
		subscribers: make(map[string]*Subscriber),
		eventChan:   make(chan Event, 100), // Buffer for events
	}

	// Start the broadcasting goroutine
	go b.run()

	return b
}

// Subscribe adds a new subscriber
func (b *Broadcaster) Subscribe(id string) *Subscriber {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove existing subscriber with same ID if exists
	if existing, exists := b.subscribers[id]; exists {
		close(existing.Channel)
	}

	sub := &Subscriber{
		ID:      id,
		Channel: make(chan Event, 10), // Buffer for subscriber
	}

	b.subscribers[id] = sub
	return sub
}

// Unsubscribe removes a subscriber
func (b *Broadcaster) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sub, exists := b.subscribers[id]; exists {
		close(sub.Channel)
		delete(b.subscribers, id)
	}
}

// Publish sends an event to all subscribers
func (b *Broadcaster) Publish(event Event) {
	event.Timestamp = time.Now()
	select {
	case b.eventChan <- event:
	default:
		// Event channel is full, drop the event
		// In production, we might want to log this
	}
}

// PublishEffectStarted publishes an effect started event
func (b *Broadcaster) PublishEffectStarted(effectName string, duration int) {
	b.Publish(Event{
		Type: EventEffectStarted,
		Data: map[string]interface{}{
			"effect":   effectName,
			"duration": duration,
		},
	})
}

// PublishEffectStopped publishes an effect stopped event
func (b *Broadcaster) PublishEffectStopped(effectName string, reason string) {
	b.Publish(Event{
		Type: EventEffectStopped,
		Data: map[string]interface{}{
			"effect": effectName,
			"reason": reason,
		},
	})
}

// PublishProgress publishes a progress event
func (b *Broadcaster) PublishProgress(effectName string, elapsed int, total int) {
	b.Publish(Event{
		Type: EventProgress,
		Data: map[string]interface{}{
			"effect":  effectName,
			"elapsed": elapsed,
			"total":   total,
		},
	})
}

// PublishDimChanged publishes a brightness change event
func (b *Broadcaster) PublishDimChanged(newLevel int) {
	b.Publish(Event{
		Type: EventDimChanged,
		Data: map[string]interface{}{
			"level": newLevel,
		},
	})
}

// PublishRawExecuted publishes a raw API execution event
func (b *Broadcaster) PublishRawExecuted(query string, result string) {
	b.Publish(Event{
		Type: EventRawExecuted,
		Data: map[string]interface{}{
			"query":  query,
			"result": result,
		},
	})
}

// PublishRingUpdate publishes a ring LED update event
func (b *Broadcaster) PublishRingUpdate(ring string, data map[string]interface{}) {
	eventData := map[string]interface{}{
		"ring": ring,
	}
	for k, v := range data {
		eventData[k] = v
	}
	b.Publish(Event{
		Type: EventRingUpdate,
		Data: eventData,
	})
}

// PublishButtonPress publishes a button press event
func (b *Broadcaster) PublishButtonPress() {
	b.Publish(Event{
		Type: EventButtonPress,
	})
}

// run is the main broadcasting loop
func (b *Broadcaster) run() {
	for event := range b.eventChan {
		b.mu.RLock()
		for _, sub := range b.subscribers {
			select {
			case sub.Channel <- event:
			default:
				// Subscriber channel is full, skip this event for this subscriber
			}
		}
		b.mu.RUnlock()
	}
}

// Close shuts down the broadcaster
func (b *Broadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Close all subscriber channels
	for _, sub := range b.subscribers {
		close(sub.Channel)
	}

	// Close the event channel
	close(b.eventChan)
}

// GetSubscriberCount returns the number of active subscribers
func (b *Broadcaster) GetSubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// ToSSEData converts an event to Server-Sent Events format
func (e Event) ToSSEData() string {
	data, _ := json.Marshal(e)
	return "data: " + string(data) + "\n\n"
}
