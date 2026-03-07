package bus

import (
	"sync"
	"testing"
	"time"
)

func TestPublishSubscribe(t *testing.T) {
	bus := New()
	defer bus.Close()

	ch := bus.Subscribe("test-topic")

	msg := InboundMessage{
		SessionID: "session1",
		Content:   "hello world",
		Role:      RoleUser,
	}

	bus.Publish("test-topic", msg)

	select {
	case received := <-ch:
		if inbound, ok := received.(InboundMessage); ok {
			if inbound.SessionID != msg.SessionID || inbound.Content != msg.Content {
				t.Fatalf("expected %+v, got %+v", msg, inbound)
			}
		} else {
			t.Fatal("received message is not InboundMessage")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := New()
	defer bus.Close()

	ch1 := bus.Subscribe("topic")
	ch2 := bus.Subscribe("topic")
	ch3 := bus.Subscribe("topic")

	msg := OutboundMessage{
		SessionID: "session1",
		Content:   "broadcast",
		Role:      RoleAssistant,
		Done:      true,
	}

	bus.Publish("topic", msg)

	var wg sync.WaitGroup
	wg.Add(3)

	checkReceiver := func(ch <-chan interface{}, name string) {
		defer wg.Done()
		select {
		case received := <-ch:
			if outbound, ok := received.(OutboundMessage); ok {
				if outbound.Content != msg.Content {
					t.Errorf("%s: expected content %q, got %q", name, msg.Content, outbound.Content)
				}
			} else {
				t.Errorf("%s: received message is not OutboundMessage", name)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("%s: timeout waiting for message", name)
		}
	}

	go checkReceiver(ch1, "sub1")
	go checkReceiver(ch2, "sub2")
	go checkReceiver(ch3, "sub3")

	wg.Wait()
}

func TestUnsubscribe(t *testing.T) {
	bus := New()
	defer bus.Close()

	ch := bus.Subscribe("topic")

	msg1 := "message1"
	bus.Publish("topic", msg1)

	select {
	case received := <-ch:
		if str, ok := received.(string); ok {
			if str != msg1 {
				t.Fatalf("expected %q, got %q", msg1, str)
			}
		} else {
			t.Fatal("received message is not string")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	bus.Unsubscribe("topic", ch)

	_, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed after unsubscribe")
	}

	msg2 := "message2"
	bus.Publish("topic", msg2)

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("received message on unsubscribed channel")
		}
	case <-time.After(100 * time.Millisecond):
	}
}

func TestConcurrentPublish(t *testing.T) {
	bus := New()
	defer bus.Close()

	ch := bus.Subscribe("concurrent-topic")

	const numPublishers = 100
	var wg sync.WaitGroup
	wg.Add(numPublishers)

	for i := 0; i < numPublishers; i++ {
		go func(n int) {
			defer wg.Done()
			msg := InboundMessage{
				SessionID: "concurrent",
				Content:   string(rune('A' + n%26)),
				Role:      RoleUser,
			}
			bus.Publish("concurrent-topic", msg)
		}(i)
	}

	wg.Wait()

	received := 0
	timeout := time.After(2 * time.Second)
	for received < numPublishers {
		select {
		case _, ok := <-ch:
			if !ok {
				t.Fatal("channel closed unexpectedly")
			}
			received++
		case <-timeout:
			t.Fatalf("timeout: received %d/%d messages", received, numPublishers)
		}
	}

	if received != numPublishers {
		t.Fatalf("expected %d messages, got %d", numPublishers, received)
	}
}

func TestClose(t *testing.T) {
	bus := New()

	ch1 := bus.Subscribe("topic1")
	ch2 := bus.Subscribe("topic2")
	ch3 := bus.Subscribe("topic1")

	bus.Close()

	_, ok1 := <-ch1
	if ok1 {
		t.Error("ch1 should be closed")
	}

	_, ok2 := <-ch2
	if ok2 {
		t.Error("ch2 should be closed")
	}

	_, ok3 := <-ch3
	if ok3 {
		t.Error("ch3 should be closed")
	}

	bus.Publish("topic1", "should not panic")

	bus.Close()
}

func TestDifferentTopics(t *testing.T) {
	bus := New()
	defer bus.Close()

	chA := bus.Subscribe("topicA")
	chB := bus.Subscribe("topicB")

	msgA := "message for A"
	msgB := "message for B"

	bus.Publish("topicA", msgA)
	bus.Publish("topicB", msgB)

	select {
	case received := <-chA:
		if str, ok := received.(string); ok {
			if str != msgA {
				t.Fatalf("topicA: expected %q, got %q", msgA, str)
			}
		}
	case <-time.After(1 * time.Second):
		t.Fatal("topicA: timeout waiting for message")
	}

	select {
	case received := <-chB:
		if str, ok := received.(string); ok {
			if str != msgB {
				t.Fatalf("topicB: expected %q, got %q", msgB, str)
			}
		}
	case <-time.After(1 * time.Second):
		t.Fatal("topicB: timeout waiting for message")
	}

	select {
	case unexpected := <-chA:
		t.Fatalf("topicA should not receive message from topicB: %v", unexpected)
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case unexpected := <-chB:
		t.Fatalf("topicB should not receive message from topicA: %v", unexpected)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestNonBlockingPublish(t *testing.T) {
	bus := New()
	defer bus.Close()

	ch := bus.Subscribe("topic")

	for i := 0; i < 100; i++ {
		bus.Publish("topic", i)
	}

	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			bus.Publish("topic", i+100)
		}
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("publish blocked when channel was full")
	}

	received := 0
	timeout := time.After(100 * time.Millisecond)
drainLoop:
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				break drainLoop
			}
			received++
		case <-timeout:
			break drainLoop
		}
	}

	if received > 100 {
		t.Logf("received %d messages (some were dropped due to full buffer)", received)
	}
}
