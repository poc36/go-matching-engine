package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/poc36/go-matching-engine/internal/market"
	"github.com/poc36/go-matching-engine/internal/orderbook"
	"github.com/poc36/go-matching-engine/internal/portfolio"
)

const (
	// PriceScale scales USD to cents for integer math.
	PriceScale = 100.0
	// SizeScale scales BTC to satoshis for integer math.
	SizeScale = 100000000.0
)

type HTTPServer struct {
	book    *orderbook.OrderBook
	history *market.HistoryManager
	port    *portfolio.Manager
	fetcher *market.PriceFetcher
}

func NewHTTPServer(book *orderbook.OrderBook, h *market.HistoryManager, p *portfolio.Manager, f *market.PriceFetcher) *HTTPServer {
	return &HTTPServer{
		book:    book,
		history: h,
		port:    p,
		fetcher: f,
	}
}

func (s *HTTPServer) handleGetDepth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	levelsStr := r.URL.Query().Get("levels")
	levels := 20
	if l, err := strconv.Atoi(levelsStr); err == nil && l > 0 {
		levels = l
	}

	userID := r.URL.Query().Get("userId")
	depth := s.book.GetDepth(levels, userID)

	// API Translation for Fractional Order support
	type LevelDTO struct {
		Price      float64 `json:"price"`
		Volume     float64 `json:"volume"`
		HasMyOrder bool    `json:"has_my_order"`
	}
	type DepthDTO struct {
		Asks []LevelDTO `json:"asks"`
		Bids []LevelDTO `json:"bids"`
	}

	dto := DepthDTO{
		Asks: make([]LevelDTO, len(depth.Asks)),
		Bids: make([]LevelDTO, len(depth.Bids)),
	}
	for i, a := range depth.Asks {
		dto.Asks[i] = LevelDTO{Price: float64(a.Price) / PriceScale, Volume: float64(a.Volume) / SizeScale, HasMyOrder: a.HasMyOrder}
	}
	for i, b := range depth.Bids {
		dto.Bids[i] = LevelDTO{Price: float64(b.Price) / PriceScale, Volume: float64(b.Volume) / SizeScale, HasMyOrder: b.HasMyOrder}
	}

	json.NewEncoder(w).Encode(dto)
}

// OrderRequest represents the JSON request from the UI
type OrderRequest struct {
	Type  string  `json:"type"` // "limit" or "market"
	Side  string  `json:"side"` // "buy" or "sell"
	Price float64 `json:"price"`
	Size  float64 `json:"size"`
}

func (s *HTTPServer) handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	side := orderbook.Buy
	if req.Side == "sell" {
		side = orderbook.Sell
	}

	orderType := orderbook.Limit
	if req.Type == "market" {
		orderType = orderbook.Market
	}

	scaledPrice := uint64(req.Price * PriceScale)
	scaledSize := uint64(req.Size * SizeScale)

	// Pre-order balance check
	isBuy := side == orderbook.Buy
	if isBuy {
		requiredUSD := req.Price * req.Size
		if req.Type == "market" {
			// Estimate market cost using live price
			priceStrs := s.fetcher.GetPrices()
			if btcPrice, err := strconv.ParseFloat(priceStrs["BTCUSDT"], 64); err == nil {
				requiredUSD = btcPrice * req.Size
			}
		}

		if !s.port.HasSufficientFunds("web-trader", isBuy, requiredUSD) {
			http.Error(w, `{"error": "Insufficient USD balance"}`, http.StatusBadRequest)
			return
		}
	} else {
		requiredBTC := req.Size
		if !s.port.HasSufficientFunds("web-trader", isBuy, requiredBTC) {
			http.Error(w, `{"error": "Insufficient BTC balance"}`, http.StatusBadRequest)
			return
		}
	}

	orderID := uuid.New().String()
	order := orderbook.NewOrder(orderID, "web-trader", side, orderType, scaledPrice, scaledSize, time.Now().UnixNano())

	trades, err := s.book.PlaceOrder(order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update Phase 6 Services
	if len(trades) > 0 {
		s.history.AddTrades(trades)

		for _, t := range trades {
			isBuy := order.Side == orderbook.Buy
			s.port.ApplyTrade(order.UserID, isBuy, t.Price, t.Size) // Apply raw uint64s to Portfolio
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"orderID": orderID,
		"trades":  len(trades),
		"status":  "SUCCESS",
	})
}

// ---- New Endpoints for Phase 6 ----

func (s *HTTPServer) handleGetTrades(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	trades := s.history.GetRecentTrades()

	type TradeDTO struct {
		Price     float64 `json:"price"`
		Size      float64 `json:"size"`
		Timestamp int64   `json:"timestamp"`
		BuyerID   string  `json:"buyer_id"`
	}

	dtos := make([]TradeDTO, len(trades))
	for i, t := range trades {
		dtos[i] = TradeDTO{
			Price:     float64(t.Price) / PriceScale,
			Size:      float64(t.Size) / SizeScale,
			Timestamp: t.Timestamp,
			BuyerID:   t.BuyerID,
		}
	}

	json.NewEncoder(w).Encode(dtos)
}

func (s *HTTPServer) handleGetPortfolio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// We use a default hardcoded user "web-trader" for demo
	b := s.port.GetBalances("web-trader")
	json.NewEncoder(w).Encode(b)
}

func (s *HTTPServer) handleGetLivePrice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	prices := s.fetcher.GetPrices()
	json.NewEncoder(w).Encode(prices)
}

func (s *HTTPServer) handleGetOpenOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	orders := s.book.GetOpenOrders(userID)

	type OrderDTO struct {
		ID        string  `json:"ID"`
		Side      string  `json:"Side"`
		Type      string  `json:"Type"`
		Price     float64 `json:"Price"`
		Size      float64 `json:"Size"`
		Remaining float64 `json:"Remaining"`
	}

	dtos := make([]OrderDTO, len(orders))
	for i, o := range orders {
		dtos[i] = OrderDTO{
			ID:        o.ID,
			Side:      string(o.Side),
			Type:      string(o.Type),
			Price:     float64(o.Price) / PriceScale,
			Size:      float64(o.Size) / SizeScale,
			Remaining: float64(o.Remaining) / SizeScale,
		}
	}

	json.NewEncoder(w).Encode(dtos)
}

func (s *HTTPServer) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID string `json:"orderId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.book.CancelOrder(req.OrderID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "SUCCESS"})
}

func (s *HTTPServer) Start(addr string) error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/depth", s.handleGetDepth)
	mux.HandleFunc("/api/order", s.handlePlaceOrder)
	mux.HandleFunc("/api/trades", s.handleGetTrades)
	mux.HandleFunc("/api/portfolio", s.handleGetPortfolio)
	mux.HandleFunc("/api/price", s.handleGetLivePrice)
	mux.HandleFunc("/api/orders", s.handleGetOpenOrders)
	mux.HandleFunc("/api/cancel", s.handleCancelOrder)

	// Serve Static Files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	return http.ListenAndServe(addr, mux)
}
