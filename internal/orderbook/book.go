package orderbook

import (
	"container/list"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Trade represents a matched execution between two orders.
type Trade struct {
	MakerOrderID string `json:"maker_order_id"`
	TakerOrderID string `json:"taker_order_id"`
	BuyerID      string `json:"buyer_id"`
	SellerID     string `json:"seller_id"`
	Price        uint64 `json:"price"`
	Size         uint64 `json:"size"`
	Timestamp    int64  `json:"timestamp"`
}

// OrderBook represents the matching engine core for a single trading pair.
type OrderBook struct {
	mu sync.RWMutex

	asks []*OrderQueue // Sorted explicitly ascending by price
	bids []*OrderQueue // Sorted explicitly descending by price

	// O(1) lookups for cancellation
	orders map[string]*list.Element
	queues map[string]*OrderQueue
}

// NewOrderBook initializes a new OrderBook.
func NewOrderBook() *OrderBook {
	return &OrderBook{
		asks:   make([]*OrderQueue, 0),
		bids:   make([]*OrderQueue, 0),
		orders: make(map[string]*list.Element),
		queues: make(map[string]*OrderQueue),
	}
}

// PlaceOrder processes an incoming order.
func (ob *OrderBook) PlaceOrder(order *Order) ([]Trade, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	if _, exists := ob.orders[order.ID]; exists {
		return nil, fmt.Errorf("order %s already exists", order.ID)
	}

	var trades []Trade
	if order.Type == Market {
		trades = ob.processMarketOrder(order)
	} else if order.Type == Limit {
		trades = ob.processLimitOrder(order)
	}

	return trades, nil
}

func (ob *OrderBook) processLimitOrder(order *Order) []Trade {
	trades := ob.matchLimitOrder(order)

	// If order is not completely filled, add remaining to book
	if !order.IsFilled() {
		ob.addOrder(order)
	}
	return trades
}

func (ob *OrderBook) processMarketOrder(order *Order) []Trade {
	// Simple matching, market orders don't go into the book
	return ob.matchLimitOrder(order) // Works identically but using limits for matching
}

func (ob *OrderBook) matchLimitOrder(order *Order) []Trade {
	var trades []Trade

	for !order.IsFilled() {
		var headQueue *OrderQueue
		if order.Side == Buy {
			if len(ob.asks) == 0 {
				break
			}
			headQueue = ob.asks[0]
			// Limit Buy order cannot match with Ask > its limit
			if order.Type == Limit && headQueue.Price > order.Price {
				break
			}
		} else { // Sell
			if len(ob.bids) == 0 {
				break
			}
			headQueue = ob.bids[0]
			// Limit Sell order cannot match with Bid < its limit
			if order.Type == Limit && headQueue.Price < order.Price {
				break
			}
		}

		headElement := headQueue.Head()
		if headElement == nil {
			ob.removeQueue(headQueue.Price, order.Side.Opposite() == Buy)
			continue
		}

		makerOrder := headElement.Value.(*Order)
		tradeSize := min(order.Remaining, makerOrder.Remaining)
		tradePrice := makerOrder.Price // Maker sets the price

		// Update order quantities
		order.Remaining -= tradeSize
		makerOrder.Remaining -= tradeSize
		headQueue.UpdateVolume(tradeSize)

		var buyerID, sellerID string
		if order.Side == Buy {
			buyerID = order.UserID
			sellerID = makerOrder.UserID
		} else {
			buyerID = makerOrder.UserID
			sellerID = order.UserID
		}

		trades = append(trades, Trade{
			MakerOrderID: makerOrder.ID,
			TakerOrderID: order.ID,
			BuyerID:      buyerID,
			SellerID:     sellerID,
			Price:        tradePrice,
			Size:         tradeSize,
			Timestamp:    time.Now().UnixNano(),
		})

		// If maker order is filled, remove it from queue
		if makerOrder.IsFilled() {
			delete(ob.orders, makerOrder.ID)
			delete(ob.queues, makerOrder.ID)
			headQueue.Remove(headElement)
			if headQueue.Len() == 0 {
				ob.removeQueue(headQueue.Price, order.Side.Opposite() == Buy)
			}
		}
	}
	return trades
}

func (ob *OrderBook) addOrder(order *Order) {
	var queue *OrderQueue

	if order.Side == Buy {
		queue = ob.findOrCreateQueue(order.Price, true)
	} else {
		queue = ob.findOrCreateQueue(order.Price, false)
	}

	e := queue.Append(order)
	ob.orders[order.ID] = e
	ob.queues[order.ID] = queue
}

func (ob *OrderBook) findOrCreateQueue(price uint64, isBid bool) *OrderQueue {
	if isBid {
		for _, q := range ob.bids {
			if q.Price == price {
				return q
			}
		}
		q := NewOrderQueue(price)
		ob.bids = append(ob.bids, q)
		sort.Slice(ob.bids, func(i, j int) bool {
			return ob.bids[i].Price > ob.bids[j].Price // descending
		})
		return q
	}

	for _, q := range ob.asks {
		if q.Price == price {
			return q
		}
	}
	q := NewOrderQueue(price)
	ob.asks = append(ob.asks, q)
	sort.Slice(ob.asks, func(i, j int) bool {
		return ob.asks[i].Price < ob.asks[j].Price // ascending
	})
	return q
}

func (ob *OrderBook) removeQueue(price uint64, isBid bool) {
	if isBid {
		for i, q := range ob.bids {
			if q.Price == price {
				ob.bids = append(ob.bids[:i], ob.bids[i+1:]...)
				return
			}
		}
	} else {
		for i, q := range ob.asks {
			if q.Price == price {
				ob.asks = append(ob.asks[:i], ob.asks[i+1:]...)
				return
			}
		}
	}
}

// CancelOrder removes an existing order from the book in O(1) time.
func (ob *OrderBook) CancelOrder(orderID string) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	e, ok := ob.orders[orderID]
	if !ok {
		return fmt.Errorf("order %s not found", orderID)
	}

	queue := ob.queues[orderID]

	// Remove from map and queue
	delete(ob.orders, orderID)
	delete(ob.queues, orderID)
	queue.Remove(e)

	// Clean up queue if empty
	targetOrder := e.Value.(*Order)
	if queue.Len() == 0 {
		ob.removeQueue(queue.Price, targetOrder.Side == Buy)
	}
	return nil
}

// Opposite returns the opposite side.
func (s Side) Opposite() Side {
	if s == Buy {
		return Sell
	}
	return Buy
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
