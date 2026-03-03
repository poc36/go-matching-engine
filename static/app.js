const API_URL = 'http://localhost:8080/api';

const elements = {
    formPrice: document.getElementById('orderPrice'),
    formSize: document.getElementById('orderSize'),
    formTotal: document.getElementById('orderTotal'),
    asksContainer: document.getElementById('asksContainer'),
    bidsContainer: document.getElementById('bidsContainer'),
    midPrice: document.getElementById('midPrice'),
    spreadValue: document.getElementById('spreadValue'),
    toastContainer: document.getElementById('toastContainer'),

    // Phase 6 UI Elements
    portUsd: document.getElementById('portUsd'),
    portBtc: document.getElementById('portBtc'),
    portTotal: document.getElementById('portTotal'),
    liveBtcPrice: document.getElementById('liveBtcPrice'),
    liveEthPrice: document.getElementById('liveEthPrice'),
    tradesContainer: document.getElementById('tradesContainer'),
    openOrdersContainer: document.getElementById('openOrdersContainer'),

    // AI Forecast Elements
    aiDirection: document.getElementById('aiDirection'),
    aiProb: document.getElementById('aiProb')
};

let currentLivePrice = 0.0;


// Calculate Local Total for Order Form
function updateTotal() {
    const price = parseFloat(elements.formPrice.value) || 0;
    const size = parseFloat(elements.formSize.value) || 0;
    elements.formTotal.innerText = (price * size).toFixed(2) + ' USD';
}

elements.formPrice.addEventListener('input', updateTotal);
elements.formSize.addEventListener('input', updateTotal);
updateTotal();

let currentOrderType = 'limit'; // Default to limit

// Toggle between Limit and Market Orders
const btnLimit = document.getElementById('btnLimit');
const btnMarket = document.getElementById('btnMarket');

btnLimit.addEventListener('click', () => {
    currentOrderType = 'limit';
    btnLimit.classList.add('active');
    btnMarket.classList.remove('active');
    elements.formPrice.disabled = false;
    updateTotal();
});

btnMarket.addEventListener('click', () => {
    currentOrderType = 'market';
    btnMarket.classList.add('active');
    btnLimit.classList.remove('active');
    elements.formPrice.disabled = true;
    elements.formPrice.value = ''; // Clear price
    elements.formTotal.innerText = 'Market Price';
});

// Click on OrderBook rows to fill form (Only if Limit)
function setOrderDetails(price, volume) {
    if (currentOrderType === 'limit') {
        elements.formPrice.value = price;
        elements.formSize.value = volume;
        updateTotal();
    } else {
        // Even in market mode, clicking depth can set the size
        elements.formSize.value = volume;
    }
}

function showToast(message, type) {
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.innerHTML = `
        <span>${message}</span>
        <span style="cursor:pointer" onclick="this.parentElement.remove()">✕</span>
    `;
    elements.toastContainer.prepend(toast);

    setTimeout(() => {
        if (toast.parentElement) toast.remove();
    }, 4000);
}

// REST: Place Order
async function placeOrder(side) {
    let finalPrice = 0;
    const finalSize = parseFloat(elements.formSize.value);

    if (currentOrderType === 'limit') {
        finalPrice = parseFloat(elements.formPrice.value);
        if (finalPrice <= 0 || isNaN(finalPrice)) {
            showToast('Invalid price. Must be > 0 for Limit orders', 'error');
            return;
        }
    }

    if (finalSize <= 0 || isNaN(finalSize)) {
        showToast('Invalid size. Must be > 0', 'error');
        return;
    }

    try {
        const response = await fetch(`${API_URL}/order`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                type: currentOrderType,
                side: side,
                price: finalPrice,
                size: finalSize
            })
        });

        if (!response.ok) {
            const err = await response.text();
            throw new Error(err);
        }

        const data = await response.json();

        const msg = data.trades > 0
            ? `Matched ${data.trades} trades instantly!`
            : `Order added to book.`;

        showToast(`Order Placed: ${data.orderID.split('-')[0]}... | ${msg}`, 'success');

        // Immediately trigger a UI refresh
        fetchAll();
    } catch (error) {
        showToast(error.message, 'error');
    }
}

// REST: View Depth (Orderbook)
async function fetchDepth() {
    try {
        const response = await fetch(`${API_URL}/depth?levels=15&userId=web-trader`);
        if (!response.ok) return;
        const data = await response.json();

        // Render Asks / Bids (Same logical rendering as before)
        let maxVolume = 0;
        const allLevels = [...(data.asks || []), ...(data.bids || [])];
        allLevels.forEach(lvl => {
            if (lvl.volume > maxVolume) maxVolume = lvl.volume;
        });

        let asksHTML = '';
        if (data.asks && data.asks.length > 0) {
            const reversedAsks = [...data.asks].reverse();
            reversedAsks.forEach(lvl => {
                const depthWidth = maxVolume > 0 ? (lvl.volume / maxVolume) * 100 : 0;
                const highlightClass = lvl.has_my_order ? 'my-order' : '';
                const displayPrice = lvl.price.toFixed(1);
                const displayVolume = lvl.volume.toFixed(4);
                const displayTotal = (lvl.price * lvl.volume).toFixed(2);

                asksHTML += `
                    <div class="book-row ask ${highlightClass}" onclick="setOrderDetails(${lvl.price}, ${lvl.volume})">
                        <div class="depth-bar" style="width: ${Math.min(depthWidth, 100)}%"></div>
                        <span class="price">${displayPrice}</span>
                        <span>${displayVolume}</span>
                        <span>${displayTotal}</span>
                    </div>
                `;
            });
        } else {
            asksHTML = '<div class="loading">No sellers</div>';
        }
        elements.asksContainer.innerHTML = asksHTML;

        let bidsHTML = '';
        if (data.bids && data.bids.length > 0) {
            data.bids.forEach(lvl => {
                const depthWidth = maxVolume > 0 ? (lvl.volume / maxVolume) * 100 : 0;
                const highlightClass = lvl.has_my_order ? 'my-order' : '';
                const displayPrice = lvl.price.toFixed(1);
                const displayVolume = lvl.volume.toFixed(4);
                const displayTotal = (lvl.price * lvl.volume).toFixed(2);

                bidsHTML += `
                    <div class="book-row bid ${highlightClass}" onclick="setOrderDetails(${lvl.price}, ${lvl.volume})">
                        <div class="depth-bar" style="width: ${Math.min(depthWidth, 100)}%"></div>
                        <span class="price">${displayPrice}</span>
                        <span>${displayVolume}</span>
                        <span>${displayTotal}</span>
                    </div>
                `;
            });
        } else {
            bidsHTML = '<div class="loading">No buyers</div>';
        }
        elements.bidsContainer.innerHTML = bidsHTML;

        // Spread
        const bestAsk = data.asks?.[0]?.price;
        const bestBid = data.bids?.[0]?.price;

        if (bestAsk !== undefined && bestBid !== undefined) {
            elements.midPrice.innerText = ((bestAsk + bestBid) / 2).toFixed(1);
            elements.spreadValue.innerText = `Spread: ${(bestAsk - bestBid).toFixed(1)}`;
        } else if (bestAsk !== undefined) {
            elements.midPrice.innerText = bestAsk;
            elements.spreadValue.innerText = 'Spread: --';
        } else if (bestBid !== undefined) {
            elements.midPrice.innerText = bestBid;
            elements.spreadValue.innerText = 'Spread: --';
        } else {
            elements.midPrice.innerText = '---';
            elements.spreadValue.innerText = 'Spread: ---';
        }
    } catch (error) {
        console.error("Depth Err", error);
    }
}

// REST: View Live Crypto Prices (Binance wrapper)
async function fetchPrice() {
    try {
        const response = await fetch(`${API_URL}/price`);
        if (!response.ok) return;

        // Data is now a map: { "BTCUSDT": "65000", "ETHUSDT": "3500", "USDTRUB": "92.5" }
        const data = await response.json();

        const btcPrice = parseFloat(data["BTCUSDT"]);
        const ethPrice = parseFloat(data["ETHUSDT"]);

        if (btcPrice > 0) {
            currentLivePrice = btcPrice;
            elements.liveBtcPrice.innerText = `$${btcPrice.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
        }

        if (ethPrice > 0) {
            elements.liveEthPrice.innerText = `$${ethPrice.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
        }
    } catch (error) {
        console.error("Price Err", error);
    }
}

// REST: View Trade History
async function fetchTrades() {
    try {
        const response = await fetch(`${API_URL}/trades`);
        if (!response.ok) return;
        const trades = await response.json();

        if (!trades || trades.length === 0) {
            elements.tradesContainer.innerHTML = '<div class="loading">No trades yet</div>';
            return;
        }

        let html = '';
        trades.forEach((t, index) => {
            // Randomly color them based on maker action (or just visually alternate for demo)
            // Ideally, the backend would tell us if it was a taker-buy or taker-sell.
            // For now, we hash the ID to decide color
            const isBuy = t.buyer_id === "web-trader";
            const sideClass = isBuy ? 'buy' : 'sell';
            const evenClass = index % 2 === 0 ? 'even' : '';

            const timeStr = new Date(t.timestamp / 1000000).toLocaleTimeString();

            html += `
                <div class="trade-row ${sideClass} ${evenClass}">
                    <span class="price">${t.price.toFixed(1)}</span>
                    <span>${t.size.toFixed(4)}</span>
                    <span class="time">${timeStr}</span>
                </div>
            `;
        });
        elements.tradesContainer.innerHTML = html;

    } catch (error) {
        console.error("Trades Err", error);
    }
}

// REST: View Portfolio
async function fetchPortfolio() {
    try {
        const response = await fetch(`${API_URL}/portfolio`);
        if (!response.ok) return;
        const data = await response.json(); // { usd: 100000.0, btc: 0.0 }

        elements.portUsd.innerText = `$${data.usd.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
        elements.portBtc.innerText = `₿${data.btc.toLocaleString(undefined, { minimumFractionDigits: 8, maximumFractionDigits: 8 })}`;

        const totalValue = data.usd + (data.btc * currentLivePrice);
        elements.portTotal.innerText = `$${totalValue.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;

    } catch (error) {
        console.error("Portfolio Err", error);
    }
}

// REST: View Open Orders
async function fetchOpenOrders() {
    try {
        const response = await fetch(`${API_URL}/orders?userId=web-trader`);
        if (!response.ok) return;
        const orders = await response.json();

        if (!orders || orders.length === 0) {
            elements.openOrdersContainer.innerHTML = '<div class="loading">No open orders</div>';
            return;
        }

        let html = '';
        orders.forEach(o => {
            const sideClass = o.Side === 'buy' ? 'side-buy' : 'side-sell';

            html += `
                <div class="open-orders-row">
                    <span class="${sideClass}">${o.Side}</span>
                    <span>${o.Type}</span>
                    <span>${o.Price.toFixed(1)}</span>
                    <span>${o.Remaining.toFixed(8)} / ${o.Size.toFixed(8)}</span>
                    <span>
                        <button class="cancel-btn" onclick="cancelOrder('${o.ID}')">Cancel</button>
                    </span>
                </div>
            `;
        });
        elements.openOrdersContainer.innerHTML = html;

    } catch (error) {
        console.error("Open Orders Err", error);
    }
}

async function cancelOrder(orderId) {
    try {
        const response = await fetch(`${API_URL}/cancel`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ orderId: orderId })
        });

        if (!response.ok) {
            const err = await response.text();
            throw new Error(err);
        }

        showToast('Order Cancelled Successfully', 'success');
        fetchAll(); // Refresh UI immediately
    } catch (error) {
        showToast(error.message, 'error');
    }
}

// Master Poller
function fetchAll() {
    fetchPrice();
    fetchDepth();
    fetchTrades();
    fetchPortfolio();
    fetchOpenOrders();
}

// Start polling
setInterval(fetchAll, 1000);
fetchAll();

// --- AI Forecast WebSocket Integration ---
function connectAIWebSocket() {
    const ws = new WebSocket('ws://localhost:8003/ws');

    ws.onopen = () => {
        console.log('Connected to ML Inference Service');
        elements.aiDirection.innerText = 'WAITING...';
        elements.aiDirection.className = 'pred-direction';
    };

    ws.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);

            elements.aiDirection.innerText = data.forecast;
            elements.aiProb.innerText = `Confidence: ${(data.confidence || (Math.max(data.probability_up, 1 - data.probability_up) * 100)).toFixed(1)}%`;

            if (data.forecast === 'UP') {
                elements.aiDirection.className = 'pred-direction up';
            } else {
                elements.aiDirection.className = 'pred-direction down';
            }
        } catch (err) {
            console.error('Error parsing AI message', err);
        }
    };

    ws.onclose = () => {
        console.log('ML WebSocket Disconnected. Reconnecting...');
        elements.aiDirection.innerText = 'OFFLINE';
        elements.aiDirection.className = 'pred-direction';
        elements.aiProb.innerText = 'Confidence: ---%';
        setTimeout(connectAIWebSocket, 2000);
    };

    ws.onerror = (err) => {
        console.error('ML WebSocket error', err);
        ws.close();
    };
}

// Initialize ML WebSocket
connectAIWebSocket();
