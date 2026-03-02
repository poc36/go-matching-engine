package market

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// binanceTickerResponse represents the JSON response from Binance API
type binanceTickerResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

// PriceFetcher periodically fetches the real-time prices of multiple assets.
type PriceFetcher struct {
	mu           sync.RWMutex
	symbols      []string
	currentPrice map[string]string
	interval     time.Duration
	stopCh       chan struct{}
}

func NewPriceFetcher(symbols []string, interval time.Duration) *PriceFetcher {
	return &PriceFetcher{
		symbols:      symbols,
		currentPrice: make(map[string]string),
		interval:     interval,
		stopCh:       make(chan struct{}),
	}
}

// Start begins the background polling.
func (f *PriceFetcher) Start() {
	go f.pollLoop()
}

// Stop cleanly stops the background polling.
func (f *PriceFetcher) Stop() {
	close(f.stopCh)
}

// GetPrices returns the latest fetched prices safely.
func (f *PriceFetcher) GetPrices() map[string]string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Create a copy to prevent concurrent map read/write if caller modifies it
	pricesCopy := make(map[string]string)
	for k, v := range f.currentPrice {
		pricesCopy[k] = v
	}
	return pricesCopy
}

func (f *PriceFetcher) pollLoop() {
	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()

	// Initial fetch
	f.fetchPrice()

	for {
		select {
		case <-ticker.C:
			f.fetchPrice()
		case <-f.stopCh:
			return
		}
	}
}

func (f *PriceFetcher) fetchPrice() {
	client := http.Client{Timeout: 5 * time.Second}
	newPrices := make(map[string]string)

	for _, sym := range f.symbols {
		url := fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", sym)
		resp, err := client.Get(url)
		if err != nil {
			slog.Error("Failed to fetch price from Binance", "symbol", sym, "err", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			slog.Error("Binance API returned non-200 status", "status", resp.Status, "symbol", sym)
			resp.Body.Close()
			continue
		}

		var data binanceTickerResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			slog.Error("Failed to decode Binance response", "err", err, "symbol", sym)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
		newPrices[sym] = data.Price
	}

	f.mu.Lock()
	for k, v := range newPrices {
		f.currentPrice[k] = v
	}
	f.mu.Unlock()
}
