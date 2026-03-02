package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/poc36/go-matching-engine/api/proto"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to gRPC server", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewExchangeMarketClient(conn)

	// Simulation Setup
	numWorkers := 10
	numOrdersPerWorker := 10000

	var totalProcessed atomic.Uint64
	var totalDuration atomic.Int64

	var wg sync.WaitGroup

	slog.Info("Starting Benchmark...", "Workers", numWorkers, "Orders/Worker", numOrdersPerWorker)
	startTime := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numOrdersPerWorker; j++ {
				// Randomly Buy or Sell
				side := pb.Side_SIDE_BUY
				if rand.Float32() > 0.5 {
					side = pb.Side_SIDE_SELL
				}

				// Price fluctuates between 90.00 and 110.00
				price := uint64(9000 + rand.Intn(2000))
				size := uint64(1 + rand.Intn(100))

				req := &pb.PlaceOrderRequest{
					UserId: fmt.Sprintf("trader-%d", workerID),
					Symbol: "BTC_USD",
					Side:   side,
					Type:   pb.OrderType_ORDER_TYPE_LIMIT,
					Price:  price,
					Size:   size,
				}

				start := time.Now()
				resp, err := client.PlaceOrder(context.Background(), req)
				duration := time.Since(start)

				if err != nil {
					slog.Error("Failed to place order", "err", err)
					continue
				}

				if resp.Status == "REJECTED" {
					slog.Warn("Order rejected", "msg", resp.Message)
				}

				totalProcessed.Add(1)
				totalDuration.Add(int64(duration))
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	totalSecs := elapsed.Seconds()
	opsPerSec := float64(totalProcessed.Load()) / totalSecs
	avgLatency := time.Duration(totalDuration.Load() / int64(totalProcessed.Load()))

	fmt.Println("\n=============================================")
	fmt.Printf("🎯 BENCHMARK RESULTS (Matching Engine)\n")
	fmt.Println("=============================================")
	fmt.Printf("Total Orders Sent:    %d\n", totalProcessed.Load())
	fmt.Printf("Total Time Elapsed:   %.2f seconds\n", totalSecs)
	fmt.Printf("Throughput (TPS):     %.0f req/sec\n", opsPerSec)
	fmt.Printf("Average Latency:      %s\n", avgLatency)
	fmt.Println("=============================================")
}
