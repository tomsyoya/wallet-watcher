# Wallet Watcher API

任意のウォレットアドレスを登録すると、Solana / Sui のトランザクション履歴を取得・保存し、API 経由で参照できるシンプルなバックエンド。

## 🚀 機能

- **/register**: アドレスを登録して監視対象に追加 ✅
- **/history**: 保存済みトランザクション履歴を取得（チェーン別・アドレス別に絞り込み可能） ✅
- **/balances**: 最新残高取得（ネイティブ通貨 + 主要トークン/コイン） ✅ **新機能**
- **/health**: ヘルスチェックで起動確認 ✅
- **バックグラウンドワーカー**: 登録済みアドレスの自動監視・データ取得 ✅
- **テストスイート**: モック・統合・API・E2Eテストを完備 ✅

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

### 残高取得

```bash
# 汎用エンドポイント
curl "http://localhost:8080/balances?chain=solana&address=${SOL_ADDR}"
curl "http://localhost:8080/balances?chain=sui&address=${SUI_ADDR}"

# チェーン専用エンドポイント
curl "http://localhost:8080/balances/solana/${SOL_ADDR}"
curl "http://localhost:8080/balances/sui/${SUI_ADDR}"
```

**レスポンス例:**
```json
{
  "address": "11111111111111111111111111111112",
  "balances": [
    {
      "token": "SOL",
      "amount": 1234567
    },
    {
      "token": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
      "amount": 5000000
    }
  ]
}
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

### API統合テスト

```bash
make test-api-balances  # /balances API の統合テスト
```

### E2Eテスト

```bash
# API サーバーが起動している状態で実行
bash test/e2e/balances_e2e_test.sh
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

## 📊 実装状況

### ✅ 実装済み (約80%)
- **API サーバー**: health, register, history, balances エンドポイント
- **バックグラウンドワーカー**: Solana/Sui 両チェーンの自動監視・データ取得
- **データベース**: 完全なスキーマ設計とマイグレーション
- **Docker環境**: 本番レディなコンテナ構成
- **テストスイート**: モック・統合・API・E2Eテスト完備
- **残高取得**: ネイティブ通貨 + トークン残高のリアルタイム取得

### ❌ 未実装 (約20%)
- **Webhook通知**: イベント発生時の自動通知機能
- **高度なエラーハンドリング**: Exponential backoff、DLQ
- **CI/CD**: GitHub Actions による自動テスト・デプロイ
- **パフォーマンス最適化**: キャッシュ、接続プール等

## 📌 今後の予定

- Webhook 通知機能の実装
- 高度なリトライ・エラーハンドリングの追加
- CI/CD パイプラインの構築
- パフォーマンス最適化