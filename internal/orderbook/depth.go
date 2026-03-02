package orderbook

// Snapshotting the Orderbook Depth

type PriceLevel struct {
	Price      uint64 `json:"price"`
	Volume     uint64 `json:"volume"`
	HasMyOrder bool   `json:"has_my_order"`
}

type Depth struct {
	Asks []PriceLevel `json:"asks"` // Ascending
	Bids []PriceLevel `json:"bids"` // Descending
}

func (ob *OrderBook) GetDepth(levels int, userID ...string) Depth {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	targetUserID := ""
	if len(userID) > 0 {
		targetUserID = userID[0]
	}

	var bids []PriceLevel
	for i := 0; i < len(ob.bids) && i < levels; i++ {
		lvl := PriceLevel{Price: ob.bids[i].Price, Volume: ob.bids[i].Volume}
		if targetUserID != "" {
			for e := ob.bids[i].Orders.Front(); e != nil; e = e.Next() {
				if e.Value.(*Order).UserID == targetUserID {
					lvl.HasMyOrder = true
					break
				}
			}
		}
		bids = append(bids, lvl)
	}

	var asks []PriceLevel
	for i := 0; i < len(ob.asks) && i < levels; i++ {
		lvl := PriceLevel{Price: ob.asks[i].Price, Volume: ob.asks[i].Volume}
		if targetUserID != "" {
			for e := ob.asks[i].Orders.Front(); e != nil; e = e.Next() {
				if e.Value.(*Order).UserID == targetUserID {
					lvl.HasMyOrder = true
					break
				}
			}
		}
		asks = append(asks, lvl)
	}

	return Depth{
		Bids: bids,
		Asks: asks,
	}
}

// GetOpenOrders returns a stable snapshot of a user's open orders.
func (ob *OrderBook) GetOpenOrders(userID string) []*Order {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	var orders []*Order
	for _, element := range ob.orders {
		order := element.Value.(*Order)
		if order.UserID == userID {
			// Return copies to prevent accidental data races
			cp := *order
			orders = append(orders, &cp)
		}
	}
	return orders
}
