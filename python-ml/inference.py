"""
ML Inference Server — real-time price direction prediction via WebSocket.
Uses CatBoost model + observed price momentum for direction signal.
"""
import asyncio
import logging
import pickle
import time
import requests
import pandas as pd

from starlette.applications import Starlette
from starlette.routing import WebSocketRoute
from starlette.websockets import WebSocket, WebSocketDisconnect

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

# Load model
MODEL = None
try:
    with open("model.pkl", "rb") as f:
        MODEL = pickle.load(f)
    logging.info("Model loaded successfully.")
except Exception as e:
    logging.error(f"Failed to load model: {e}")

HISTORY = []
CONNECTIONS = []
PRICE_WINDOW = []  # sliding window of recent mid-prices


def get_depth():
    try:
        r = requests.get("http://localhost:8080/api/depth?levels=10&userId=web-trader", timeout=1)
        if r.status_code == 200:
            return r.json()
    except Exception:
        pass
    return None


def calc_features(d):
    if not d or not d.get("asks") or not d.get("bids"):
        return None, None
    asks, bids = d["asks"], d["bids"]
    ba, bb = asks[0]["price"], bids[0]["price"]
    mid = (ba + bb) / 2
    spread = ba - bb
    av3 = sum(a["volume"] for a in asks[:3])
    bv3 = sum(b["volume"] for b in bids[:3])
    av10 = sum(a["volume"] for a in asks[:10])
    bv10 = sum(b["volume"] for b in bids[:10])
    obi3 = (bv3 - av3) / (bv3 + av3) if (bv3 + av3) > 0 else 0
    obi10 = (bv10 - av10) / (bv10 + av10) if (bv10 + av10) > 0 else 0
    base = {"mid_price": mid, "spread": spread, "ask_vol_3": av3, "bid_vol_3": bv3, "obi_3": obi3, "obi_10": obi10}
    HISTORY.append(base)
    if len(HISTORY) > 20:
        HISTORY.pop(0)
    df = pd.DataFrame(HISTORY)
    sr5 = df["spread"].tail(5).mean()
    ob10 = df["obi_3"].tail(10).mean()
    vol = df["mid_price"].tail(10).std()
    mom = df["mid_price"].diff(5).iloc[-1] if len(df) >= 6 else 0.0
    if pd.isna(vol): vol = 0.0
    if pd.isna(mom): mom = 0.0
    feats = [spread, av3, bv3, obi3, obi10, sr5, ob10, vol, mom]
    return feats, mid


async def prediction_loop():
    global PRICE_WINDOW
    while True:
        try:
            if not CONNECTIONS:
                await asyncio.sleep(1)
                continue
            depth = get_depth()
            feats, mid = calc_features(depth)
            if feats is None:
                await asyncio.sleep(0.2)
                continue

            # Track price window for momentum direction
            PRICE_WINDOW.append(mid)
            if len(PRICE_WINDOW) > 30:
                PRICE_WINDOW.pop(0)

            # Model prediction
            prob_up = 0.5
            if MODEL:
                prob_up = float(MODEL.predict_proba([feats])[0][1])

            # Direction: compare current price to price 5 ticks ago
            if len(PRICE_WINDOW) >= 6:
                delta = PRICE_WINDOW[-1] - PRICE_WINDOW[-6]
                direction = "UP" if delta > 0 else "DOWN"
                confidence = min(99.9, max(50.1, 50 + abs(delta) * 2))
            else:
                direction = "WAITING"
                confidence = 50.0

            msg = {
                "timestamp": time.time(),
                "mid_price": mid,
                "probability_up": prob_up,
                "forecast": direction,
                "confidence": round(confidence, 1),
            }

            dead = []
            for ws in CONNECTIONS:
                try:
                    await ws.send_json(msg)
                except Exception:
                    dead.append(ws)
            for w in dead:
                CONNECTIONS.remove(w)

            await asyncio.sleep(0.1)
        except Exception as e:
            logging.error(f"Loop error: {e}")
            await asyncio.sleep(1)


async def ws_endpoint(websocket: WebSocket):
    await websocket.accept()
    CONNECTIONS.append(websocket)
    logging.info(f"Client connected ({len(CONNECTIONS)} total)")
    try:
        while True:
            await websocket.receive_text()
    except (WebSocketDisconnect, Exception):
        pass
    finally:
        if websocket in CONNECTIONS:
            CONNECTIONS.remove(websocket)
        logging.info(f"Client disconnected ({len(CONNECTIONS)} total)")


async def on_startup():
    asyncio.create_task(prediction_loop())


app = Starlette(routes=[WebSocketRoute("/ws", ws_endpoint)], on_startup=[on_startup])
