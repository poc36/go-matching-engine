package orderbook

// Side represents the side of an order (Buy or Sell).
type Side string

const (
	Buy  Side = "buy"
	Sell Side = "sell"
)

// OrderType represents the type of an order.
type OrderType string

const (
	Limit  OrderType = "limit"
	Market OrderType = "market"
)

// Order represents a single order in the matching engine.
// We use uint64 for Price and Size to avoid floating point inaccuracies.
// The consumer of this API is responsible for applying the correct decimal multiplier.
type Order struct {
	ID        string
	UserID    string
	Side      Side
	Type      OrderType
	Price     uint64
	Size      uint64
	Remaining uint64
	Timestamp int64
}

// NewOrder creates a new Order instance.
func NewOrder(id, userID string, side Side, orderType OrderType, price, size uint64, timestamp int64) *Order {
	return &Order{
		ID:        id,
		UserID:    userID,
		Side:      side,
		Type:      orderType,
		Price:     price,
		Size:      size,
		Remaining: size,
		Timestamp: timestamp,
	}
}

// IsFilled returns true if the order has been completely filled.
func (o *Order) IsFilled() bool {
	return o.Remaining == 0
}
