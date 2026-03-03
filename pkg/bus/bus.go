package bus

import "sync"

// Bus defines the interface for a publish-subscribe message bus
type Bus interface {
	Publish(topic string, msg interface{})
	Subscribe(topic string) <-chan interface{}
	Unsubscribe(topic string, ch <-chan interface{})
	Close()
}

// memBus is an in-memory implementation of Bus
type memBus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan interface{}
	closed      bool
}

// New creates a new in-memory message bus
func New() Bus {
	return &memBus{
		subscribers: make(map[string][]chan interface{}),
	}
}

// Publish sends a message to all subscribers of a topic (non-blocking)
func (b *memBus) Publish(topic string, msg interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	subs, ok := b.subscribers[topic]
	if !ok {
		return
	}

	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

// Subscribe creates a new subscription channel for a topic
func (b *memBus) Subscribe(topic string) <-chan interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan interface{}, 100)
	b.subscribers[topic] = append(b.subscribers[topic], ch)
	return ch
}

// Unsubscribe removes a subscription channel from a topic
func (b *memBus) Unsubscribe(topic string, ch <-chan interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs, ok := b.subscribers[topic]
	if !ok {
		return
	}

	for i, sub := range subs {
		if sub == ch {
			close(sub)
			b.subscribers[topic] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

// Close shuts down the bus and closes all subscriber channels
func (b *memBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true

	for _, subs := range b.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}

	b.subscribers = make(map[string][]chan interface{})
}
