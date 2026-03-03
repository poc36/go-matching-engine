import time
import requests
import pandas as pd
import numpy as np
import os
import argparse
from datetime import datetime
import logging

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

def get_depth(base_url, symbol="BTCUSDT"):
    try:
        # UserID web-trader is hardcoded in Go engine
        resp = requests.get(f"{base_url}/api/depth?levels=10&userId=web-trader", timeout=1)
        if resp.status_code == 200:
            return resp.json()
    except Exception as e:
        logging.error(f"Depth fetch error: {e}")
    return None

def get_live_price(base_url, symbol="BTCUSDT"):
    try:
        resp = requests.get(f"{base_url}/api/price", timeout=1)
        if resp.status_code == 200:
            data = resp.json()
            if symbol in data:
               return float(data[symbol])
    except Exception as e:
        logging.error(f"Price fetch error: {e}")
    return None

def calculate_features(depth_data, live_price):
    if not depth_data or not depth_data.get('asks') or not depth_data.get('bids'):
        return None
        
    asks =  depth_data['asks']
    bids =  depth_data['bids']
    
    # Best ask and bid
    best_ask = asks[0]['price']
    best_bid = bids[0]['price']
    
    # 1. Spread (разница между лучшей ценой продажи и покупки)
    spread = best_ask - best_bid
    
    # 2. Mid-Price (средняя цена)
    mid_price = (best_ask + best_bid) / 2
    
    # Calculate volume up to N levels
    ask_vol_3 = sum(a['volume'] for a in asks[:3])
    bid_vol_3 = sum(b['volume'] for b in bids[:3])
    
    ask_vol_10 = sum(a['volume'] for a in asks[:10])
    bid_vol_10 = sum(b['volume'] for b in bids[:10])

    # 3. Order Book Imbalance (OBI) - Дисбаланс ордеров (кто сильнее давит - покупатели или продавцы)
    # Формула: (BidVolume - AskVolume) / (BidVolume + AskVolume)
    # Диапазон: [-1; 1]. Ближе к 1 - сильное давление покупателей (ожидаем рост цены)
    obi_3 = (bid_vol_3 - ask_vol_3) / (bid_vol_3 + ask_vol_3) if (bid_vol_3 + ask_vol_3) > 0 else 0
    obi_10 = (bid_vol_10 - ask_vol_10) / (bid_vol_10 + ask_vol_10) if (bid_vol_10 + ask_vol_10) > 0 else 0
    
    # 4. Micro-Price (взвешенная по объему средняя цена)
    # Дает лучшее представление об "истинной" цене, чем Mid-Price
    micro_price = mid_price
    if (bid_vol_3 + ask_vol_3) > 0:
        micro_price = (best_ask * bid_vol_3 + best_bid * ask_vol_3) / (bid_vol_3 + ask_vol_3)

    return {
        'timestamp': time.time(),
        'best_ask': best_ask,
        'best_bid': best_bid,
        'mid_price': mid_price,
        'micro_price': micro_price,
        'spread': spread,
        'ask_vol_3': ask_vol_3,
        'bid_vol_3': bid_vol_3,
        'obi_3': obi_3,
        'obi_10': obi_10,
        'external_price': live_price if live_price else mid_price
    }

def main():
    parser = argparse.ArgumentParser(description='Go Matching Engine Data Collector')
    parser.add_argument('--url', type=str, default='http://localhost:8080', help='Go Engine HTTP API URL')
    parser.add_argument('--hz', type=int, default=10, help='Polling frequency in Hz (requests per second)')
    parser.add_argument('--output', type=str, default='dataset.csv', help='Output CSV file')
    args = parser.parse_args()
    
    logging.info(f"Starting collector. Polling {args.url} at {args.hz}Hz. Saving to {args.output}")
    
    # Create empty CSV with headers if it doesn't exist
    if not os.path.exists(args.output):
        df_empty = pd.DataFrame(columns=[
            'timestamp', 'best_ask', 'best_bid', 'mid_price', 'micro_price', 
            'spread', 'ask_vol_3', 'bid_vol_3', 'obi_3', 'obi_10', 'external_price'
        ])
        df_empty.to_csv(args.output, index=False)
    
    sleep_time = 1.0 / args.hz
    batch_data = []
    last_save = time.time()
    save_interval = 5.0 # Save to disk every 5 seconds
    
    try:
        while True:
            start_t = time.time()
            
            # Fetch data (в реальном HFT мы бы получали это по WebSockets, но для пет-проекта polling - супер метод)
            live_price = get_live_price(args.url)
            depth = get_depth(args.url)
            
            features = calculate_features(depth, live_price)
            if features:
                batch_data.append(features)
            
            # Save batch to disk 
            if time.time() - last_save > save_interval:
                if batch_data:
                    df = pd.DataFrame(batch_data)
                    df.to_csv(args.output, mode='a', header=False, index=False)
                    logging.info(f"Saved {len(batch_data)} rows. Total time elapsed: {time.time() - last_save:.1f}s. MidPrice: {features['mid_price']}")
                    batch_data = []
                    last_save = time.time()
            
            # Sleep to maintain Hz
            elapsed = time.time() - start_t
            if elapsed < sleep_time:
                time.sleep(sleep_time - elapsed)
                
    except KeyboardInterrupt:
        logging.info("Collector stopped by user.")
        # Save remaining
        if batch_data:
            pd.DataFrame(batch_data).to_csv(args.output, mode='a', header=False, index=False)
            logging.info(f"Saved final {len(batch_data)} rows.")

if __name__ == "__main__":
    main()
