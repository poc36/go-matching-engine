package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	pb "github.com/poc36/go-matching-engine/api/proto"
	"github.com/poc36/go-matching-engine/internal/market"
	"github.com/poc36/go-matching-engine/internal/orderbook"
	"github.com/poc36/go-matching-engine/internal/portfolio"
	"github.com/poc36/go-matching-engine/internal/pubsub"
)

// ExchangeServer implementing the gRPC ExchangeMarket service.
type ExchangeServer struct {
	pb.UnimplementedExchangeMarketServer
	book    *orderbook.OrderBook
	pubsub  pubsub.EventPublisher
	history *market.HistoryManager
	port    *portfolio.Manager
}

// NewExchangeServer creates a new instance of ExchangeServer.
func NewExchangeServer(book *orderbook.OrderBook, publisher pubsub.EventPublisher, h *market.HistoryManager, p *portfolio.Manager) *ExchangeServer {
	return &ExchangeServer{
		book:    book,
		pubsub:  publisher,
		history: h,
		port:    p,
	}
}

// PlaceOrder handles incoming gRPC requests to place an order.
func (s *ExchangeServer) PlaceOrder(ctx context.Context, req *pb.PlaceOrderRequest) (*pb.OrderResponse, error) {
	if req.Size == 0 || req.Price == 0 {
		return nil, fmt.Errorf("size and price must be greater than zero")
	}

	side := orderbook.Buy
	if req.Side == pb.Side_SIDE_SELL {
		side = orderbook.Sell
	} else if req.Side == pb.Side_SIDE_UNSPECIFIED {
		return nil, fmt.Errorf("side must be specified")
	}

	orderType := orderbook.Limit
	if req.Type == pb.OrderType_ORDER_TYPE_MARKET {
		orderType = orderbook.Market
	}

	orderID := uuid.New().String()
	order := orderbook.NewOrder(
		orderID,
		req.UserId,
		side,
		orderType,
		req.Price,
		req.Size,
		time.Now().UnixNano(),
	)

	trades, err := s.book.PlaceOrder(order)
	if err != nil {
		slog.Error("Failed to place order", "err", err, "orderID", orderID)
		return &pb.OrderResponse{
			OrderId: orderID,
			Status:  "REJECTED",
			Message: err.Error(),
		}, nil
	}

	slog.Info("Order placed successfully", "orderID", orderID, "trades_executed", len(trades))

	// Update Trade History, Portfolio, and Publish to Redis
	if len(trades) > 0 {
		s.history.AddTrades(trades)

		for _, t := range trades {
			// In our simplified model, Maker and Taker sides correspond to who placed the order.
			isBuy := order.Side == orderbook.Buy
			s.port.ApplyTrade(order.UserID, isBuy, t.Price, t.Size)
		}

		go func(matchedTrades []orderbook.Trade) {
			pubErr := s.pubsub.PublishTrades(context.Background(), req.Symbol, matchedTrades)
			if pubErr != nil {
				slog.Error("Failed to publish trades asynchronously", "err", pubErr)
			}
		}(trades)
	}

	status := "OPEN"
	if order.IsFilled() {
		status = "FILLED"
	} else if len(trades) > 0 {
		status = "PARTIAL"
	}

	return &pb.OrderResponse{
		OrderId: orderID,
		Status:  status,
		Message: fmt.Sprintf("Order placed. Matched %d trades immediately.", len(trades)),
	}, nil
}

// CancelOrder handles incoming cancellation requests.
func (s *ExchangeServer) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelResponse, error) {
	err := s.book.CancelOrder(req.OrderId)
	if err != nil {
		return &pb.CancelResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	slog.Info("Order cancelled successfully", "orderID", req.OrderId)
	return &pb.CancelResponse{
		Success: true,
		Message: "Order cancelled successfully.",
	}, nil
}
