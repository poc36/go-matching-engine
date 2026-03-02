package portfolio

import (
	"fmt"
	"sync"
)

// Balances represents the asset holdings. We use float64 here for UI simplicity,
// although a real system would use decimal/uint64.
type Balances struct {
	USD float64 `json:"usd"`
	BTC float64 `json:"btc"`
}

// Manager handles the virtual balances of users.
type Manager struct {
	mu       sync.RWMutex
	balances map[string]*Balances
}

func NewManager() *Manager {
	return &Manager{
		balances: make(map[string]*Balances),
	}
}

// GetBalances returns the balances for a user.
func (m *Manager) GetBalances(userID string) Balances {
	m.mu.RLock()
	defer m.mu.RUnlock()

	b, ok := m.balances[userID]
	if !ok {
		// Mock initial balance for new users (e.g., $100k demo money, 0 BTC)
		return Balances{USD: 100000.0, BTC: 0.0}
	}
	return *b
}

// HasSufficientFunds checks if the user has enough of the base or quote asset to place an order.
func (m *Manager) HasSufficientFunds(userID string, isBuy bool, amount float64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	b, ok := m.balances[userID]
	if !ok {
		// New users have $100k demo money
		if isBuy {
			return 100000.0 >= amount
		}
		return 0.0 >= amount
	}

	if isBuy {
		return b.USD >= amount // amount is in USD required
	}
	return b.BTC >= amount // amount is in BTC required
}

// ApplyTrade adjusts the balances based on a matched trade.
func (m *Manager) ApplyTrade(userID string, isBuy bool, price uint64, size uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.balances[userID]
	if !ok {
		// Initialize with demo money
		b = &Balances{USD: 100000.0, BTC: 0.0}
		m.balances[userID] = b
	}

	// Calculate float amounts using scales
	tradeAmountBTC := float64(size) / 100000000.0 // SizeScale
	tradePriceUSD := float64(price) / 100.0       // PriceScale
	tradeValueUSD := tradeAmountBTC * tradePriceUSD

	if isBuy {
		b.USD -= tradeValueUSD
		b.BTC += tradeAmountBTC
		fmt.Printf("[Portfolio] %s bought %f BTC for %f USD\n", userID, tradeAmountBTC, tradeValueUSD)
	} else {
		b.USD += tradeValueUSD
		b.BTC -= tradeAmountBTC
		fmt.Printf("[Portfolio] %s sold %f BTC for %f USD\n", userID, tradeAmountBTC, tradeValueUSD)
	}
}
