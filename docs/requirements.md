# Wallet Watcher API — 要件まとめ
## 🎯 目的

- Solana / Sui のアドレスを登録すると、トランザクション履歴を自動で取得・保存し、API から参照できるようにする

- Web3 コミュニティでバックエンド実装の理解を深めるための学習・実戦プロジェクト

## 🏗 アーキテクチャ概要

- 言語: Go

- データベース: PostgreSQL

- メッセージング: （現状未導入、将来的に Redis Streams / SQS を想定）

-デプロイ: Docker Compose (api, worker, postgres)

##📦 コンポーネント
### 1. API サーバ (/api)

- エンドポイント

    - GET /health : 起動確認

    - POST /register : アドレスをチェーン別に登録

    - GET /history : 登録済みアドレスのトランザクション履歴取得

    - クエリ: chain, address, limit, before

レスポンス

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

- 対応済みチェーン: Solana

- 設定（環境変数）

    - SOLANA_RPC_URL : Solana RPC エンドポイント

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

- tx_events_sui

    - 同上（Sui 用）

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

### 🚧 今後の拡張予定

- Sui ワーカー実装（チェックポイント追跡）

- 残高取得 API /balances

- Webhook 通知（入金/特定イベント時にPOST）

- 冪等性強化（DLQ, 再試行戦略）

- CI/CD 統合（GitHub Actions でテスト自動化）