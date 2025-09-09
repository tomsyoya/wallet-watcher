# Wallet Watcher API — 要件まとめ

## 🎯 目的
- Solana / Sui のアドレスを登録すると、トランザクション履歴を自動で取得・保存し、API から参照できるようにする  
- Web3 コミュニティでバックエンド実装の理解を深めるための学習・実戦プロジェクト  

---

## 🏗 アーキテクチャ概要
- 言語: Go
- データベース: PostgreSQL
- デプロイ: Docker Compose (api, worker, postgres)
- メッセージング: 将来的に Redis Streams / SQS を想定

---

## 📦 コンポーネント

### 1. API サーバ (`/api`)
- **エンドポイント**
  - `GET /health` : 起動確認
  - `POST /register` : アドレスをチェーン別に登録
  - `GET /history` : 登録済みアドレスのトランザクション履歴取得  
    - クエリ: `chain`, `address`, `limit`, `before`
  - `GET /balances` : 最新残高取得（ネイティブ通貨 + 主要トークン/コイン） ← **新規追加**
  - `POST /webhook/register` : Webhook URL 登録 ← **新規追加**
  - `DELETE /webhook/register` : Webhook URL 削除 ← **新規追加**

- **レスポンス例 (`/history`)**

  ```json
  {
    "events": [
      {
        "tx_hash": "xxx",
        "ts": "2025-08-28T12:34:56Z",
        "sender": "...",
        "receiver": "...",
        "token": "SOL",
        "amount": 1000,
        "fee": 5000,
        "method": "transfer"
      }
    ],
    "next_before": "2025-08-27T23:59:59Z"
  }
```

### 2. ワーカー (/worker)

- バックグラウンドで登録済みアドレスをポーリング

- 新規 Tx を RPC 経由で取得 → 正規化して DB に保存 → カーソル更新

- チェーン対応

    - Solana : 実装済み

    - Sui : Checkpoints 経由でブロック確定後にイベントを取得 ← 追加仕様

- 設定（環境変数）

    - SOLANA_RPC_URL : Solana RPC エンドポイント

    - SUI_RPC_URL : Sui RPC エンドポイント

    - POLL_INTERVAL_SEC : ポーリング間隔（デフォルト 5 秒）

    - BATCH_SIZE : 1回あたり取得件数（デフォルト 10）

### 3. データベーススキーマ

- watched_addresses_solana

    - address (PK)

    - last_slot (カーソル)

    - created_at, updated_at

- watched_addresses_sui

    - address (PK)

    - last_checkpoint (カーソル)

    - created_at, updated_at

- tx_events_solana

    - tx_hash, ts, sender, receiver, token, amount, fee, method, raw

    - PK : (tx_hash, ts)

- tx_events_sui

    - tx_hash, ts, sender, receiver, token, amount, fee, method, raw

    - PK : (tx_hash, ts)

- webhook_subscriptions

    - id (PK, serial)

    - url

    - chain

    - address

    - event_type

    - created_at

### 4. マイグレーション

- make migrate で migrations/*.sql をソートして順次適用

- マウント: ./migrations:/migrations:ro

### 5. テスト

- test/solana/ : Solana 用統合テスト

- test/sui/ : Sui 用統合テスト

- 特徴

    - テスト直前: migrations テーブルの直近範囲をバックアップ → 対象行を削除

    - テスト中: テスト用データを INSERT → /history API 経由で検証

    - テスト終了後: バックアップから復元 → データ残らない

- 実行方法

```bash
make build-test-image
make test-integration-solana
make test-integration-sui
```

### 6. Docker / Compose

- Dockerfile multi-stage (build → distroless runtime)

- イメージに /api /worker の両バイナリを内包

- Compose サービス

    - api: /api を起動

    - worker: /worker を起動

    - postgres: DB

    - ボリューム: pgdata:/var/lib/postgresql/data

## 🚀 未実装機能の仕様

### 1. Sui ワーカー

* **方式**: Checkpoint を順に追跡 (`getCheckpointSummary`, `getTransactionBlock`)
* **カーソル**: `last_checkpoint` を保存し、そこから未処理の分を取得
* **処理内容**: Tx イベントを正規化して `tx_events_sui` に保存

---

### 2. /balances API

* **エンドポイント**: `GET /balances?chain={solana|sui}&address=...`
* **レスポンス例**

  ```json
  {
    "address": "...",
    "balances": [
      { "token": "SOL", "amount": 1234567 },
      { "token": "USDC", "amount": 5000000 }
    ]
  }
  ```
* **内部処理**

  * Solana: `getBalance` + `getTokenAccountsByOwner`
  * Sui: `getBalance(s)` + `getCoins`

---

### 3. Webhook 通知

* **目的**: 新規 Tx、または特定の条件（例: 入金、特定メソッド呼び出し）で通知
* **設定**: `webhook_subscriptions` にアドレスと URL を登録
* **通知形式**

  ```json
  {
    "address": "...",
    "chain": "solana",
    "event": {
      "tx_hash": "...",
      "amount": 1234,
      "token": "SOL",
      "ts": "2025-09-09T12:00:00Z"
    }
  }
  ```
* **配信方式**: worker がイベント検知時に HTTP POST
* **再試行**: 3 回まで指数バックオフ、失敗は DLQ に保存（将来 Redis/SQS で拡張）

---

### 4. 冪等性とリトライ

* **挿入**: `ON CONFLICT (tx_hash, ts) DO NOTHING` により重複防止
* **RPC 失敗**: Exponential backoff で再試行
* **DLQ**: 再試行上限に達したイベントは別テーブルに保存（`failed_events`）

---

### 5. CI/CD

* **テスト自動化**: GitHub Actions

  * `make build-test-image`
  * `make test-integration-solana` / `make test-integration-sui`
* **マイグレーション検証**: CI 内で `make migrate` → テーブル定義チェック
* **デプロイ**: 将来的に Fly.io / Render / GCP へ
