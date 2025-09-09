# Wallet Watcher API

任意のウォレットアドレスを登録すると、Solana / Sui のトランザクション履歴を取得・保存し、API 経由で参照できるシンプルなバックエンド。

## 🚀 機能

- /register: アドレスを登録して監視対象に追加

- /history: 保存済みトランザクション履歴を取得（チェーン別・アドレス別に絞り込み可能）

- ヘルスチェック: /health で起動確認

- テスト: Solana / Sui それぞれのモックテスト・インテグレーションテストを用意

## 🛠 前提

- Docker / Docker Compose / make

- .env に各種設定を記載

### .env の例

```.env
APP_PORT=8080
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=walletwatcher
DATABASE_URL=postgres://postgres:postgres@postgres:5432/walletwatcher?sslmode=disable

# Solana RPC
SOLANA_RPC_URL=https://api.mainnet-beta.solana.com
SOL_ADDR=<取引があるSolanaアドレス>

# Sui RPC
SUI_RPC_URL=https://fullnode.mainnet.sui.io:443
SUI_ADDR=0x<取引があるSuiアドレス>
```

## 📦 初回セットアップ

```bash
# 起動
make up

# 初期マイグレーション
make migrate FILE=0001_init.sql
make migrate FILE=0002_chain_split.sql
```

## ✅ API 動作確認

### ヘルスチェック

```bash
curl http://localhost:8080/health
# => ok
```

### 監視対象アドレス登録

```bash
# Solana
curl -s -X POST http://localhost:8080/register \
  -H 'Content-Type: application/json' \
  -d "{\"chain\":\"solana\",\"address\":\"${SOL_ADDR}\"}"

# Sui（
curl -s -X POST http://localhost:8080/register \
  -H 'Content-Type: application/json' \
  -d "{\"chain\":\"sui\",\"address\":\"${SUI_ADDR}\"}"
```

### 履歴取得

```bash
# 最新10件（Solana全体）
curl "http://localhost:8080/history?chain=solana&limit=10"

# 指定アドレスの履歴（Sui）
curl "http://localhost:8080/history?chain=sui&address=0x<SUI_ADDRESS>&limit=20"

# ページング（next_beforeを利用）
curl "http://localhost:8080/history?chain=solana&before=2025-08-28T23:59:59Z&limit=10"
```


## 🧪 テスト

### モックテスト

```bash
make test-solana   # Solana モック
make test-sui      # Sui モック
```
### インテグレーションテスト（実ノード + Postgres）

.env に RPC URL / アドレスを設定済みであれば実行可能。

```bash
make test-integration-solana
make test-integration-sui
```

### 環境変数の確認（デバッグ用）

```bash
make test-env
```

## ⚙️ データベース

- watched_addresses_solana / watched_addresses_sui
  - 登録済みアドレスとカーソルを保持

- tx_events_solana / tx_events_sui
  - トランザクション履歴（正規化済み）

マイグレーションは migrations/ に保存。

## 📌 今後の予定

- バックグラウンドワーカーでアドレスごとの差分取得 & 自動挿入

- トークン転送やコール内容の詳細解析

- CI/CD にテスト統合