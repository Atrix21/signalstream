package sse

import (
	"testing"
	"time"
)

func TestBroker_SubscribeAndBroadcast(t *testing.T) {
	broker := NewBroker()
	userID := "user-123"

	ch := broker.Subscribe(userID)

	broker.Broadcast(userID, map[string]string{"msg": "hello"})

	select {
	case data := <-ch:
		if len(data) == 0 {
			t.Fatal("received empty data")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestBroker_UnsubscribeStopsDelivery(t *testing.T) {
	broker := NewBroker()
	userID := "user-456"

	ch := broker.Subscribe(userID)
	broker.Unsubscribe(userID, ch)

	// Channel should be closed after unsubscribe.
	_, ok := <-ch
	if ok {
		t.Fatal("channel should be closed after unsubscribe")
	}
}

func TestBroker_MultipleClients(t *testing.T) {
	broker := NewBroker()
	userID := "user-789"

	ch1 := broker.Subscribe(userID)
	ch2 := broker.Subscribe(userID)

	broker.Broadcast(userID, "test")

	for _, ch := range []chan []byte{ch1, ch2} {
		select {
		case data := <-ch:
			if len(data) == 0 {
				t.Fatal("received empty data")
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for broadcast on one of the clients")
		}
	}
}

func TestBroker_IsolationBetweenUsers(t *testing.T) {
	broker := NewBroker()

	ch1 := broker.Subscribe("user-a")
	ch2 := broker.Subscribe("user-b")

	broker.Broadcast("user-a", "only for user-a")

	select {
	case <-ch1:
		// expected
	case <-time.After(time.Second):
		t.Fatal("user-a should receive their broadcast")
	}

	select {
	case <-ch2:
		t.Fatal("user-b should not receive user-a's broadcast")
	case <-time.After(50 * time.Millisecond):
		// expected: no message
	}
}

func TestBroker_BroadcastToNonexistentUser(t *testing.T) {
	broker := NewBroker()
	// Should not panic.
	broker.Broadcast("nonexistent-user", "hello")
}
