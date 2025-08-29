-- ================================
-- 0002_chain_split.sql (修正版)
-- アドレス監視テーブルをチェーン別に分割（統合案：カーソル列を同居）
-- tx_events_* もチェーン別に作成
-- ================================

-- 1) 監視アドレス（Solana）
CREATE TABLE IF NOT EXISTS watched_addresses_solana (
  id             BIGSERIAL PRIMARY KEY,
  address        TEXT NOT NULL,
  -- カーソル（最後に処理したsignature）
  last_signature TEXT,
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(address)
);

-- 2) 監視アドレス（Sui）
CREATE TABLE IF NOT EXISTS watched_addresses_sui (
  id              BIGSERIAL PRIMARY KEY,
  address         TEXT NOT NULL,
  -- カーソル（最後に処理したcheckpoint）
  last_checkpoint BIGINT,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(address)
);

-- 3) 既存の旧テーブルが存在する場合の移行（OPTIONAL）
-- ※ 旧テーブル(registrations, cursors)が残っている環境のみ実行される
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'registrations') THEN
    INSERT INTO watched_addresses_solana (address, created_at, updated_at)
    SELECT r.address, COALESCE(r.created_at, now()), now()
    FROM registrations r
    WHERE r.chain = 'solana'
    ON CONFLICT (address) DO NOTHING;

    INSERT INTO watched_addresses_sui (address, created_at, updated_at)
    SELECT r.address, COALESCE(r.created_at, now()), now()
    FROM registrations r
    WHERE r.chain = 'sui'
    ON CONFLICT (address) DO NOTHING;
  END IF;

  IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'cursors') THEN
    -- Solana: last_marker を last_signature に
    UPDATE watched_addresses_solana s
    SET last_signature = c.last_marker,
        updated_at = now()
    FROM cursors c
    WHERE c.chain = 'solana' AND c.address = s.address AND c.last_marker IS NOT NULL;

    -- Sui: last_marker を数値にできる場合のみ反映
    UPDATE watched_addresses_sui s
    SET last_checkpoint = NULLIF(c.last_marker, '')::BIGINT,
        updated_at = now()
    FROM cursors c
    WHERE c.chain = 'sui' AND c.address = s.address
      AND c.last_marker ~ '^[0-9]+$';
  END IF;
END$$;

-- 4) 正規化イベント（チェーン別）
CREATE TABLE IF NOT EXISTS tx_events_solana (
  id         BIGSERIAL PRIMARY KEY,
  tx_hash    TEXT NOT NULL,
  ts         TIMESTAMPTZ NOT NULL,
  sender     TEXT,
  receiver   TEXT,
  token      TEXT,
  amount     NUMERIC(78,0),
  fee        NUMERIC(78,0),
  method     TEXT,
  raw        JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tx_hash, ts)
);

CREATE TABLE IF NOT EXISTS tx_events_sui (
  id         BIGSERIAL PRIMARY KEY,
  tx_hash    TEXT NOT NULL,
  ts         TIMESTAMPTZ NOT NULL,
  sender     TEXT,
  receiver   TEXT,
  token      TEXT,
  amount     NUMERIC(78,0),
  fee        NUMERIC(78,0),
  method     TEXT,
  raw        JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tx_hash, ts)
);

-- 5) 代表的な索引
CREATE INDEX IF NOT EXISTS idx_tx_solana_ts        ON tx_events_solana (ts DESC);
CREATE INDEX IF NOT EXISTS idx_tx_solana_sender_ts ON tx_events_solana (sender, ts DESC);
CREATE INDEX IF NOT EXISTS idx_tx_solana_recv_ts   ON tx_events_solana (receiver, ts DESC);

CREATE INDEX IF NOT EXISTS idx_tx_sui_ts        ON tx_events_sui (ts DESC);
CREATE INDEX IF NOT EXISTS idx_tx_sui_sender_ts ON tx_events_sui (sender, ts DESC);
CREATE INDEX IF NOT EXISTS idx_tx_sui_recv_ts   ON tx_events_sui (receiver, ts DESC);
