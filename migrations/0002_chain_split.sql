-- 0002_chain_split.sql
-- 互換維持用の補正（既存環境で 0001 以前に作られたズレを吸収）
-- 何度流しても安全

-- 旧テーブル／旧制約が存在していた場合に備えた補正（例示的に IF EXISTS で防御）
-- ここでは実データ破壊を避け、存在確認のみ or 追補に留める

-- tx_events_solana の一意性（古い環境で無い場合のみ追加）
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid = 'tx_events_solana'::regclass
      AND (contype = 'p' OR (contype = 'u' AND conkey = ARRAY[
        (SELECT attnum FROM pg_attribute WHERE attrelid='tx_events_solana'::regclass AND attname='tx_hash'),
        (SELECT attnum FROM pg_attribute WHERE attrelid='tx_events_solana'::regclass AND attname='ts')
      ]))
  ) THEN
    ALTER TABLE tx_events_solana
      ADD CONSTRAINT uq_tx_events_solana_hash_ts UNIQUE (tx_hash, ts);
  END IF;
END $$;

-- tx_events_sui の一意性
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid = 'tx_events_sui'::regclass
      AND (contype = 'p' OR (contype = 'u' AND conkey = ARRAY[
        (SELECT attnum FROM pg_attribute WHERE attrelid='tx_events_sui'::regclass AND attname='tx_hash'),
        (SELECT attnum FROM pg_attribute WHERE attrelid='tx_events_sui'::regclass AND attname='ts')
      ]))
  ) THEN
    ALTER TABLE tx_events_sui
      ADD CONSTRAINT uq_tx_events_sui_hash_ts UNIQUE (tx_hash, ts);
  END IF;
END $$;

-- インデックスの取りこぼしを補完（IF NOT EXISTS）
CREATE INDEX IF NOT EXISTS idx_tx_solana_ts          ON tx_events_solana (ts DESC);
CREATE INDEX IF NOT EXISTS idx_tx_solana_sender_ts   ON tx_events_solana (sender, ts DESC);
CREATE INDEX IF NOT EXISTS idx_tx_solana_receiver_ts ON tx_events_solana (receiver, ts DESC);

CREATE INDEX IF NOT EXISTS idx_tx_sui_ts          ON tx_events_sui (ts DESC);
CREATE INDEX IF NOT EXISTS idx_tx_sui_sender_ts   ON tx_events_sui (sender, ts DESC);
CREATE INDEX IF NOT EXISTS idx_tx_sui_receiver_ts ON tx_events_sui (receiver, ts DESC);
