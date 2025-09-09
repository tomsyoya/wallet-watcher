-- 0001_init.sql
-- ベーススキーマ（Solana / Sui それぞれの監視アドレス & 取引イベント）
-- 何度流しても安全なように IF NOT EXISTS を徹底

-- ===========================
-- watched_addresses_solana
-- ===========================
CREATE TABLE IF NOT EXISTS watched_addresses_solana (
  address     text PRIMARY KEY,
  last_slot   bigint,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

-- ===========================
-- watched_addresses_sui
-- ===========================
CREATE TABLE IF NOT EXISTS watched_addresses_sui (
  address         text PRIMARY KEY,
  last_checkpoint bigint,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);

-- ===========================
-- tx_events_solana
-- ===========================
CREATE TABLE IF NOT EXISTS tx_events_solana (
  tx_hash   text        NOT NULL,
  ts        timestamptz NOT NULL,
  sender    text,
  receiver  text,
  token     text,
  amount    numeric(78,0),
  fee       numeric(78,0),
  method    text,
  raw       jsonb,
  CONSTRAINT pk_tx_events_solana PRIMARY KEY (tx_hash, ts)
);

-- カラム型の補正（既存環境で型がズレている場合に備えて）
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'tx_events_solana' AND column_name = 'amount' AND udt_name <> 'numeric'
  ) THEN
    ALTER TABLE tx_events_solana ALTER COLUMN amount TYPE numeric(78,0) USING amount::numeric;
  END IF;
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'tx_events_solana' AND column_name = 'fee' AND udt_name <> 'numeric'
  ) THEN
    ALTER TABLE tx_events_solana ALTER COLUMN fee TYPE numeric(78,0) USING fee::numeric;
  END IF;
END $$;

-- インデックス
CREATE INDEX IF NOT EXISTS idx_tx_solana_ts
  ON tx_events_solana (ts DESC);
CREATE INDEX IF NOT EXISTS idx_tx_solana_sender_ts
  ON tx_events_solana (sender, ts DESC);
CREATE INDEX IF NOT EXISTS idx_tx_solana_receiver_ts
  ON tx_events_solana (receiver, ts DESC);

-- ===========================
-- tx_events_sui
-- ===========================
CREATE TABLE IF NOT EXISTS tx_events_sui (
  tx_hash   text        NOT NULL,
  ts        timestamptz NOT NULL,
  sender    text,
  receiver  text,
  token     text,
  amount    numeric(78,0),
  fee       numeric(78,0),
  method    text,
  raw       jsonb,
  CONSTRAINT pk_tx_events_sui PRIMARY KEY (tx_hash, ts)
);

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'tx_events_sui' AND column_name = 'amount' AND udt_name <> 'numeric'
  ) THEN
    ALTER TABLE tx_events_sui ALTER COLUMN amount TYPE numeric(78,0) USING amount::numeric;
  END IF;
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'tx_events_sui' AND column_name = 'fee' AND udt_name <> 'numeric'
  ) THEN
    ALTER TABLE tx_events_sui ALTER COLUMN fee TYPE numeric(78,0) USING fee::numeric;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_tx_sui_ts
  ON tx_events_sui (ts DESC);
CREATE INDEX IF NOT EXISTS idx_tx_sui_sender_ts
  ON tx_events_sui (sender, ts DESC);
CREATE INDEX IF NOT EXISTS idx_tx_sui_receiver_ts
  ON tx_events_sui (receiver, ts DESC);
