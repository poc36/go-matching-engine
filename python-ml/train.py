import pandas as pd
import numpy as np
from catboost import CatBoostClassifier
from sklearn.model_selection import train_test_split
from sklearn.metrics import accuracy_score, precision_score, roc_auc_score
import mlflow
import os
import argparse
import logging
import pickle

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

def create_features_and_targets(file_path, forecast_horizon=10):
    logging.info(f"Loading data from {file_path}")
    df = pd.read_csv(file_path)
    if 'best_ask' not in df.columns: # fallback if headers missing
        df.columns = [
            'timestamp', 'best_ask', 'best_bid', 'mid_price', 'micro_price', 
            'spread', 'ask_vol_3', 'bid_vol_3', 'obi_3', 'obi_10', 'external_price'
        ]
    
    # Target definition: Will the mid-price go up in the next N ticks?
    # We shift the mid_price backwards by horizon (i.e. future price)
    df['future_mid_price'] = df['mid_price'].shift(-forecast_horizon)
    # 1 if price went up, 0 if price went down or stayed same
    df['target'] = (df['future_mid_price'] > df['mid_price']).astype(int)
    
    # Feature Engineering (Rolling Windows simulating SQL window functions)
    df['spread_rolling_5'] = df['spread'].rolling(window=5).mean()
    df['obi_3_rolling_10'] = df['obi_3'].rolling(window=10).mean()
    df['volatility_10'] = df['mid_price'].rolling(window=10).std()
    df['price_momentum'] = df['mid_price'].diff(5)
    
    # Drop rows with NaNs caused by shift/rolling
    df.dropna(inplace=True)
    
    features = [
        'spread', 'ask_vol_3', 'bid_vol_3', 'obi_3', 'obi_10', 
        'spread_rolling_5', 'obi_3_rolling_10', 'volatility_10', 'price_momentum'
    ]
    
    return df[features], df['target']

def train_model(X, y):
    # Train-test split (chronological for time series)
    split_idx = int(len(X) * 0.8)
    X_train, X_test = X.iloc[:split_idx], X.iloc[split_idx:]
    y_train, y_test = y.iloc[:split_idx], y.iloc[split_idx:]
    
    logging.info(f"Training on {len(X_train)} samples, testing on {len(X_test)} samples.")
    
    with mlflow.start_run():
        params = {
            'iterations': 200,
            'learning_rate': 0.1,
            'depth': 6,
            'loss_function': 'Logloss',
            'verbose': 50
        }
        
        mlflow.log_params(params)
        
        model = CatBoostClassifier(**params)
        model.fit(X_train, y_train, eval_set=(X_test, y_test))
        
        # Predictions
        y_pred = model.predict(X_test)
        y_prob = model.predict_proba(X_test)[:, 1]
        
        # Metrics
        acc = accuracy_score(y_test, y_pred)
        prec = precision_score(y_test, y_pred, zero_division=0)
        try:
            auc = roc_auc_score(y_test, y_prob)
        except ValueError:
            auc = 0.5 # fallback if only one class exists in small synthetics
            
        logging.info(f"Model Metrics - Acc: {acc:.4f}, Prec: {prec:.4f}, AUC: {auc:.4f}")
        
        mlflow.log_metric("accuracy", acc)
        mlflow.log_metric("precision", prec)
        mlflow.log_metric("roc_auc", auc)
        
        # Save model
        with open("model.pkl", "wb") as f:
            pickle.dump(model, f)
        
        mlflow.log_artifact("model.pkl")
        logging.info("Model saved to model.pkl and logged to MLflow.")

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--input', type=str, default='dataset.csv', help='Input CSV dataset')
    args = parser.parse_args()
    
    # Set up MLflow
    mlflow.set_tracking_uri("sqlite:///mlruns.db")
    mlflow.set_experiment("HFT_MicroPrice_Prediction")
    
    X, y = create_features_and_targets(args.input)
    
    # Need at least some valid rows
    if len(X) < 50:
         logging.error(f"Not enough data to train. Got {len(X)} valid rows. Make sure dataset.csv has enough data.")
         return
         
    train_model(X, y)

if __name__ == "__main__":
    main()
