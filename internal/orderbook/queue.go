package orderbook

import "container/list"

// OrderQueue represents a list of orders at a specific price level.
// It maintains the total volume of all orders in the queue to allow O(1) lookups of the depth.
type OrderQueue struct {
	Price  uint64
	Volume uint64
	Orders *list.List
}

// NewOrderQueue creates a new OrderQueue for a specific price.
func NewOrderQueue(price uint64) *OrderQueue {
	return &OrderQueue{
		Price:  price,
		Volume: 0,
		Orders: list.New(),
	}
}

// Append adds an order to the end of the queue (Price-Time Priority).
func (q *OrderQueue) Append(o *Order) *list.Element {
	q.Volume += o.Remaining
	return q.Orders.PushBack(o)
}

// Remove deletes an order from the queue and updates the total volume.
func (q *OrderQueue) Remove(e *list.Element) *Order {
	if e == nil {
		return nil
	}

	o, ok := e.Value.(*Order)
	if !ok {
		return nil
	}

	q.Orders.Remove(e)
	q.Volume -= o.Remaining
	return o
}

// UpdateVolume is called when an order is partially executed to subtract from the total queue volume.
func (q *OrderQueue) UpdateVolume(matchedQuantity uint64) {
	if q.Volume >= matchedQuantity {
		q.Volume -= matchedQuantity
	} else {
		q.Volume = 0
	}
}

// Head returns the first element in the queue (the oldest order).
func (q *OrderQueue) Head() *list.Element {
	return q.Orders.Front()
}

// Len returns the number of orders in the queue.
func (q *OrderQueue) Len() int {
	return q.Orders.Len()
}
