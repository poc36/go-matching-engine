package orderbook

import (
	"testing"
)

func TestOrderBook_AddAndMatchLimitOrders(t *testing.T) {
	ob := NewOrderBook()

	// 1. Add Bids (Buyers)
	o1 := NewOrder("1", "u1", Buy, Limit, 100, 50, 1) // Bid at 100
	o2 := NewOrder("2", "u2", Buy, Limit, 90, 100, 2) // Bid at 90
	o3 := NewOrder("3", "u3", Buy, Limit, 100, 20, 3) // Bid at 100 (later than o1)

	ob.PlaceOrder(o1)
	ob.PlaceOrder(o2)
	ob.PlaceOrder(o3)

	if len(ob.bids) != 2 {
		t.Errorf("Expected 2 bid price levels, got %d", len(ob.bids))
	}
	if ob.bids[0].Price != 100 {
		t.Errorf("Expected best bid to be 100, got %d", ob.bids[0].Price)
	}
	if ob.bids[0].Volume != 70 { // 50 + 20
		t.Errorf("Expected volume at price 100 to be 70, got %d", ob.bids[0].Volume)
	}

	// 2. Add an Ask (Seller) that matches partially
	// Selling 60 at price 95. Should match with o1 (50) and o3 (10). o2 (price 90) is untouched limit-wise.
	tradeOrder := NewOrder("4", "u4", Sell, Limit, 95, 60, 4)
	trades, err := ob.PlaceOrder(tradeOrder)

	if err != nil {
		t.Fatal(err)
	}

	if len(trades) != 2 {
		t.Errorf("Expected 2 trades, got %d", len(trades))
	}

	// First trade should be against o1
	if trades[0].MakerOrderID != "1" {
		t.Errorf("Expected first trade maker to be '1', got %s", trades[0].MakerOrderID)
	}
	if trades[0].Size != 50 {
		t.Errorf("Expected first trade size to be 50, got %d", trades[0].Size)
	}
	if trades[0].Price != 100 {
		t.Errorf("Expected first trade price to be 100, got %d", trades[0].Price)
	}

	// Second trade should be against o3
	if trades[1].MakerOrderID != "3" {
		t.Errorf("Expected second trade maker to be '3', got %s", trades[1].MakerOrderID)
	}
	if trades[1].Size != 10 {
		t.Errorf("Expected second trade size to be 10, got %d", trades[1].Size)
	}
	if trades[1].Price != 100 {
		t.Errorf("Expected second trade price to be 100, got %d", trades[1].Price)
	}

	// Check post-match state
	if tradeOrder.Remaining != 0 {
		t.Errorf("Expected taker order to be fully filled, remaining: %d", tradeOrder.Remaining)
	}
	if len(ob.asks) != 0 {
		t.Errorf("Expected 0 ask levels (fully filled), got %d", len(ob.asks))
	}
	// Bid level 100 should still have 10 remaining from o3
	if ob.bids[0].Volume != 10 {
		t.Errorf("Expected volume at price 100 to be 10, got %d", ob.bids[0].Volume)
	}
}

func TestOrderBook_CancelOrder(t *testing.T) {
	ob := NewOrderBook()

	o1 := NewOrder("1", "u1", Buy, Limit, 100, 50, 1)
	ob.PlaceOrder(o1)

	if len(ob.bids) != 1 {
		t.Errorf("Expected 1 bid level")
	}

	err := ob.CancelOrder("1")
	if err != nil {
		t.Errorf("Cancel failed: %v", err)
	}

	if len(ob.bids) != 0 {
		t.Errorf("Expected 0 bid levels after cancellation, got %d", len(ob.bids))
	}

	if _, ok := ob.orders["1"]; ok {
		t.Error("Order 1 still present in map after cancellation")
	}
}
