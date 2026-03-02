package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/poc36/go-matching-engine/internal/orderbook"
	"github.com/redis/go-redis/v9"
)

type EventPublisher interface {
	PublishTrades(ctx context.Context, symbol string, trades []orderbook.Trade) error
}

type RedisPublisher struct {
	client *redis.Client
}

func NewRedisPublisher(addr string) *RedisPublisher {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisPublisher{
		client: rdb,
	}
}

func (p *RedisPublisher) PublishTrades(ctx context.Context, symbol string, trades []orderbook.Trade) error {
	if len(trades) == 0 {
		return nil
	}

	channel := fmt.Sprintf("trades:%s", symbol)

	// In a high frequency system, we would batch these or use binary encoding (protobuf).
	// Here we use JSON for simplicity and readability by clients.
	for _, trade := range trades {
		data, err := json.Marshal(trade)
		if err != nil {
			slog.Error("Failed to marshal trade", "err", err)
			continue
		}

		err = p.client.Publish(ctx, channel, data).Err()
		if err != nil {
			slog.Error("Failed to publish trade to Redis", "err", err)
		}
	}

	slog.Info("Published trades to Redis", "count", len(trades), "channel", channel)
	return nil
}

// DummyPublisher is used when Redis is not available or during testing.
type DummyPublisher struct{}

func NewDummyPublisher() *DummyPublisher {
	return &DummyPublisher{}
}

func (p *DummyPublisher) PublishTrades(ctx context.Context, symbol string, trades []orderbook.Trade) error {
	if len(trades) > 0 {
		slog.Info("[Dummy PubSub] Trades executed", "count", len(trades), "symbol", symbol)
	}
	return nil
}
