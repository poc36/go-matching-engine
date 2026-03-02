package orderbook

import (
	"testing"
)

func TestOrderQueueAppend(t *testing.T) {
	q := NewOrderQueue(100)
	o1 := NewOrder("1", "user1", Buy, Limit, 100, 10, 1)
	o2 := NewOrder("2", "user2", Buy, Limit, 100, 20, 2)

	_ = q.Append(o1)
	_ = q.Append(o2)

	if q.Len() != 2 {
		t.Errorf("Expected queue length to be 2, got %d", q.Len())
	}
	if q.Volume != 30 {
		t.Errorf("Expected queue volume to be 30, got %d", q.Volume)
	}

	// Verify Price-Time priority (first in, first out)
	head := q.Head()
	if head.Value.(*Order).ID != "1" {
		t.Errorf("Expected head order ID to be '1', got '%s'", head.Value.(*Order).ID)
	}
}

func TestOrderQueueRemove(t *testing.T) {
	q := NewOrderQueue(200)
	o1 := NewOrder("1", "user1", Sell, Limit, 200, 50, 1)
	e1 := q.Append(o1)

	if q.Volume != 50 {
		t.Errorf("Expected queue volume to be 50, got %d", q.Volume)
	}

	removedOrder := q.Remove(e1)
	if removedOrder.ID != "1" {
		t.Errorf("Expected removed order ID to be '1', got '%s'", removedOrder.ID)
	}
	if q.Len() != 0 {
		t.Errorf("Expected queue length to be 0, got %d", q.Len())
	}
	if q.Volume != 0 {
		t.Errorf("Expected queue volume to be 0, got %d", q.Volume)
	}
}

func TestOrderQueueUpdateVolume(t *testing.T) {
	q := NewOrderQueue(300)
	q.Volume = 100

	q.UpdateVolume(30)
	if q.Volume != 70 {
		t.Errorf("Expected volume 70, got %d", q.Volume)
	}

	q.UpdateVolume(100)
	if q.Volume != 0 {
		t.Errorf("Expected volume 0, got %d", q.Volume)
	}
}
