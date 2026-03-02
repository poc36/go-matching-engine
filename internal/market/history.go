package market

import (
	"sync"

	"github.com/poc36/go-matching-engine/internal/orderbook"
)

// HistoryManager keeps track of the latest executed trades for the public feed.
type HistoryManager struct {
	mu     sync.RWMutex
	trades []orderbook.Trade
	limit  int
}

// NewHistoryManager creates a new HistoryManager storing up to 'limit' trades.
func NewHistoryManager(limit int) *HistoryManager {
	return &HistoryManager{
		trades: make([]orderbook.Trade, 0, limit),
		limit:  limit,
	}
}

// AddTrades appends new matched trades to the history.
func (h *HistoryManager) AddTrades(newTrades []orderbook.Trade) {
	if len(newTrades) == 0 {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Prepend new trades to keep the list ordered from newest to oldest
	h.trades = append(newTrades, h.trades...)

	// Truncate to limit
	if len(h.trades) > h.limit {
		h.trades = h.trades[:h.limit]
	}
}

// GetRecentTrades returns a copy of the recent trades safely.
func (h *HistoryManager) GetRecentTrades() []orderbook.Trade {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return a copy to prevent concurrent modification issues
	cpy := make([]orderbook.Trade, len(h.trades))
	copy(cpy, h.trades)
	return cpy
}
