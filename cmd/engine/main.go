package main

import (
	"log/slog"
	"net"
	"os"

	"time"

	"google.golang.org/grpc"

	pb "github.com/poc36/go-matching-engine/api/proto"
	"github.com/poc36/go-matching-engine/internal/market"
	"github.com/poc36/go-matching-engine/internal/orderbook"
	"github.com/poc36/go-matching-engine/internal/portfolio"
	"github.com/poc36/go-matching-engine/internal/pubsub"
	"github.com/poc36/go-matching-engine/internal/server"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Initialize Core Matching Engine OrderBook
	book := orderbook.NewOrderBook()

	// Initialize Redis Publisher (or Dummy if no Redis URL provided)
	redisAddr := os.Getenv("REDIS_ADDR") // e.g. "localhost:6379"
	var publisher pubsub.EventPublisher
	if redisAddr != "" {
		publisher = pubsub.NewRedisPublisher(redisAddr)
		slog.Info("Connected to Redis mapping", "addr", redisAddr)
	} else {
		publisher = pubsub.NewDummyPublisher()
		slog.Info("Using Dummy PubSub (REDIS_ADDR not set)")
	}

	// Phase 6: Initialize Live Data & Portfolio Managers
	historyManager := market.NewHistoryManager(50)
	portfolioManager := portfolio.NewManager()

	// Create Price Fetcher for BTC, ETH, and RUB
	symbols := []string{"BTCUSDT", "ETHUSDT", "USDTRUB"}
	priceFetcher := market.NewPriceFetcher(symbols, 2*time.Second)
	priceFetcher.Start()
	defer priceFetcher.Stop()

	marketMaker := market.NewMarketMaker(book, priceFetcher, historyManager, portfolioManager, 3*time.Second)
	marketMaker.Start()
	defer marketMaker.Stop()

	// Initialize gRPC Server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	exchangeServer := server.NewExchangeServer(book, publisher, historyManager, portfolioManager)

	pb.RegisterExchangeMarketServer(grpcServer, exchangeServer)

	// Start Web Server
	webServer := server.NewHTTPServer(book, historyManager, portfolioManager, priceFetcher)
	go func() {
		slog.Info("Web Trading Terminal serving on http://localhost:8080")
		if err := webServer.Start(":8080"); err != nil {
			slog.Error("HTTP server failed", "error", err)
		}
	}()

	slog.Info("gRPC Matching Engine starting on", "port", 50051)
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}
