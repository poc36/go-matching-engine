# ⚡ Go P2P Matching Engine

[English](#english) | [Русский](#русский)

---

<a name="english"></a>
## 🇺🇸 English

A high-performance, real-time trading engine built with **Go**, **gRPC**, and **Redis**, featuring a stunning Cyberpunk-themed Web Terminal.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.24.1-00ADD8.svg?logo=go)

### 🚀 Overview
This project is a decentralized-ready P2P Matching Engine designed for high throughput and low latency. It handles Limit and Market orders, tracks real-time global prices, and provides a fully immersive trading experience directly in your browser.

### ✨ Key Features
- **High-Performance Core**: Internal matching engine using efficient data structures for Limit and Market orders.
- **Real-Time Data**: Integrated `PriceFetcher` that pulls live BTC/ETH prices directly from Binance.
- **Market Maker Bot**: Automated liquidity provider that maintains a realistic spread around live market prices.
- **Cyberpunk Web Terminal**:
  - **CRT Visuals**: Scanlines, grid overlays, and rhythmic glows for a true retro-future feel.
  - **Precision Trading**: 8-decimal internal precision for BTC with a clean 4-decimal UI display.
- **Portfolio Management**: Demo balances with built-in protection against negative balances.
- **Scalable Architecture**: decoupled components communicating via gRPC and Redis Pub/Sub.

### 🛠 Tech Stack
- **Backend**: Go (Golang)
- **Communication**: gRPC (Protocol Buffers)
- **Data/Events**: Redis (Pub/Sub for real-time updates)
- **Frontend**: Vanilla JS, CSS3 (Cyberpunk Design System)

### 📦 Installation & Setup
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

<a name="русский"></a>
## 🇷🇺 Русский

Высокопроизводительный торговый движок в реальном времени, построенный на **Go**, **gRPC** и **Redis**, с потрясающим веб-терминалом в стиле Киберпанк.

### 🚀 Обзор
Этот проект представляет собой децентрализованный P2P движок сопоставления ордеров (Matching Engine), разработанный для высокой пропускной способности и низкой задержки. Он обрабатывает лимитные и рыночные ордера, отслеживает мировые цены в реальном времени и обеспечивает полное погружение в торговлю прямо в вашем браузере.

### ✨ Ключевые особенности
- **Высокопроизводительное ядро**: Внутренний механизм сопоставления, использующий эффективные структуры данных.
- **Данные в реальном времени**: Интегрированный `PriceFetcher`, который подтягивает актуальные цены BTC/ETH напрямую с Binance.
- **Бот Маркет-мейкер**: Автоматический провайдер ликвидности, который поддерживает реалистичный спред вокруг рыночных цен.
- **Киберпанк Веб-терминал**:
  - **CRT Визуал**: Эффект сканирующих линий, сетки и неонового свечения (ретро-футуризм).
  - **Точность торговли**: Внутренняя точность 8 знаков для BTC с чистым отображением 4 знаков в интерфейсе.
- **Управление портфелем**: Демо-балансы с встроенной защитой от отрицательного остатка.
- **Масштабируемая архитектура**: Разделенные компоненты, взаимодействующие через gRPC и Redis Pub/Sub.

### 🛠 Технологический стек
- **Бэкенд**: Go (Golang)
- **Связь**: gRPC (Protocol Buffers)
- **Данные/События**: Redis (Pub/Sub для обновлений в реальном времени)
- **Фронтенд**: Vanilla JS, CSS3 (Cyberpunk Design System)

### 📦 Установка и запуск
1. **Клонируйте репозиторий**:
   ```bash
   git clone https://github.com/poc36/go-matching-engine.git
   cd go-matching-engine
   ```
2. **Загрузите зависимости**:
   ```bash
   go mod download
   ```
3. **Запустите движок**:
   ```bash
   go run cmd/engine/main.go
   ```
4. **Откройте терминал**:
   Перейдите по адресу `http://localhost:8080` в вашем браузере.

---

## 📄 License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
