# ⚡ Go P2P Matching Engine

A high-performance, real-time trading engine built with **Go**, **gRPC**, and **Redis**, featuring a stunning Cyberpunk-themed Web Terminal.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.24.1-00ADD8.svg?logo=go)

---

## 🚀 Overview

This project is a decentralized-ready P2P Matching Engine designed for high throughput and low latency. It handles Limit and Market orders, tracks real-time global prices, and provides a fully immersive trading experience directly in your browser.

### ✨ Key Features

- **High-Performance Core**: Internal matching engine using efficient data structures for Limit and Market orders.
- **Real-Time Data**: Integrated `PriceFetcher` that pulls live BTC/ETH prices directly from Binance.
- **Market Maker Bot**: Automated liquidity provider that maintains a realistic spread around live market prices.
- **Cyberpunk Web Terminal**:
  - **CRT Visuals**: Scanlines, grid overlays, and rhythmic glows for a true retro-future feel.
  - **Procedural Audio**: Real-time synthesized sound effects (clicks, swooshes, alerts) using the Web Audio API.
  - **Precision Trading**: 8-decimal internal precision for BTC with a clean 4-decimal UI display.
- **Portfolio Management**: Demo balances with built-in protection against negative balances.
- **Scalable Architecture**: decoupled components communicating via gRPC and Redis Pub/Sub.

---

## 🛠 Tech Stack

- **Backend**: Go (Golang)
- **Communication**: gRPC (Protocol Buffers)
- **Data/Events**: Redis (Pub/Sub for real-time updates)
- **Frontend**: Vanilla JS, CSS3 (Cyberpunk Design System), HTML5 Web Audio API

---

## 📦 Installation & Setup

### Prerequisites

- [Go](https://golang.org/doc/install) 1.24+
- [Redis](https://redis.io/download) (optional, uses dummy pubsub if not detected)

### Running the Engine

1. **Clone the repository**:
   ```bash
   git clone https://github.com/poc36/go-matching-engine.git
   cd go-matching-engine
   ```

2. **Download dependencies**:
   ```bash
   go mod download
   ```

3. **Start the Engine**:
   ```bash
   go run cmd/engine/main.go
   ```

4. **Open the Terminal**:
   Navigate to `http://localhost:8080` in your browser.

---

## 🖥 API Reference

The engine exposes a REST API on port `8080`:

- `POST /api/order`: Place a new Limit or Market order.
- `GET /api/depth`: Get current Order Book depth.
- `GET /api/trades`: View recent trade history.
- `GET /api/portfolio`: Check demo balances.
- `POST /api/cancel`: Cancel an open order.

---

## 🎨 UI & Aesthetics

The web terminal is designed with a "High Tech, Low Life" philosophy. 
- **Row Clicking**: Click any row in the Order Book to instantly populate the trade form.
- **Dynamic Spread**: Watch the spread oscillate in real-time as the Market Maker reacts to global price shifts.

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
