package market

import (
	"log/slog"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/poc36/go-matching-engine/internal/orderbook"
	"github.com/poc36/go-matching-engine/internal/portfolio"
)

// MarketMaker bot periodically places orders around the live market price.
type MarketMaker struct {
	mu           sync.Mutex
	book         *orderbook.OrderBook
	fetcher      *PriceFetcher
	history      *HistoryManager
	port         *portfolio.Manager
	activeOrders []string
	interval     time.Duration
	stopCh       chan struct{}
}

// NewMarketMaker creates a new market maker bot instance.
func NewMarketMaker(book *orderbook.OrderBook, fetcher *PriceFetcher, history *HistoryManager, port *portfolio.Manager, interval time.Duration) *MarketMaker {
	// Seed random generator for order sizes
	rand.Seed(time.Now().UnixNano())

	return &MarketMaker{
		book:         book,
		fetcher:      fetcher,
		history:      history,
		port:         port,
		interval:     interval,
		activeOrders: make([]string, 0),
		stopCh:       make(chan struct{}),
	}
}

// Start begins the market maker loop.
func (m *MarketMaker) Start() {
	go m.loop()
}

// Stop cleanly shuts down the market maker.
func (m *MarketMaker) Stop() {
	close(m.stopCh)
}

func (m *MarketMaker) loop() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Initial market making
	m.makeMarket()

	for {
		select {
		case <-ticker.C:
			m.makeMarket()
		case <-m.stopCh:
			m.cancelAllOrders()
			return
		}
	}
}

func (m *MarketMaker) cancelAllOrders() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range m.activeOrders {
		// Ignore error since order might have been fully filled and removed
		_ = m.book.CancelOrder(id)
	}
	// Clear the list of active orders
	m.activeOrders = make([]string, 0)
}

func (m *MarketMaker) makeMarket() {
	prices := m.fetcher.GetPrices()
	priceStr := prices["BTCUSDT"]
	if priceStr == "0.00" || priceStr == "" {
		return // Wait for the first valid price fetch
	}

	priceFloat, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		slog.Error("MarketMaker failed to parse live price", "price", priceStr, "err", err)
		return
	}

	basePrice := uint64(priceFloat)
	if basePrice == 0 {
		return
	}

	// Remove previously placed but unfilled orders
	m.cancelAllOrders()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Parameters for our liquidity
	spreadFloat := 2.0
	levels := 10 // How many bid/ask levels to create

	for i := 1; i <= levels; i++ {
		// Generate random volume between 0.1 and 1.5 BTC roughly
		askSizeFloat := (rand.Float64() * 1.4) + 0.1
		bidSizeFloat := (rand.Float64() * 1.4) + 0.1

		askPriceFloat := priceFloat + float64(i)*spreadFloat
		bidPriceFloat := priceFloat - float64(i)*spreadFloat

		askPrice := uint64(askPriceFloat * 100.0) // PriceScale
		bidPrice := uint64(bidPriceFloat * 100.0)

		askSize := uint64(askSizeFloat * 100000000.0) // SizeScale
		bidSize := uint64(bidSizeFloat * 100000000.0)

		askID := uuid.New().String()
		bidID := uuid.New().String()

		askOrder := orderbook.NewOrder(askID, "market-maker", orderbook.Sell, orderbook.Limit, askPrice, askSize, time.Now().UnixNano())
		bidOrder := orderbook.NewOrder(bidID, "market-maker", orderbook.Buy, orderbook.Limit, bidPrice, bidSize, time.Now().UnixNano())

		// Place ask
		tradesAsk, errAsk := m.book.PlaceOrder(askOrder)
		if errAsk == nil {
			m.activeOrders = append(m.activeOrders, askID)
			if len(tradesAsk) > 0 {
				m.history.AddTrades(tradesAsk)
				m.applyPortfolioTrades("market-maker", orderbook.Sell, tradesAsk)
			}
		}

		// Place bid
		tradesBid, errBid := m.book.PlaceOrder(bidOrder)
		if errBid == nil {
			m.activeOrders = append(m.activeOrders, bidID)
			if len(tradesBid) > 0 {
				m.history.AddTrades(tradesBid)
				m.applyPortfolioTrades("market-maker", orderbook.Buy, tradesBid)
			}
		}
	}

	// Ensure the portfolio user exists if not already
	// (Market Maker could accumulate huge negative sums depending on the test cases, but it's a dummy for liquidity)
}

func (m *MarketMaker) applyPortfolioTrades(userID string, side orderbook.Side, trades []orderbook.Trade) {
	for _, t := range trades {
		isBuy := side == orderbook.Buy
		m.port.ApplyTrade(userID, isBuy, t.Price, t.Size)
	}
}
