-- チェーン種別
CREATE TYPE chain AS ENUM ('solana', 'sui');

-- ウォレット登録
CREATE TABLE IF NOT EXISTS registrations (
  id            BIGSERIAL PRIMARY KEY,
  chain         chain NOT NULL,
  address       TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(chain, address)
);

-- 取得カーソル（アドレスごと）
CREATE TABLE IF NOT EXISTS cursors (
  id            BIGSERIAL PRIMARY KEY,
  chain         chain NOT NULL,
  address       TEXT NOT NULL,
  -- Solana: last_signature、Sui: last_checkpoint などを柔軟に保持
  last_marker   TEXT,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(chain, address)
);

-- 正規化トランザクション（最小）
CREATE TABLE IF NOT EXISTS tx_events (
  id            BIGSERIAL PRIMARY KEY,
  chain         chain NOT NULL,
  tx_hash       TEXT NOT NULL,
  ts            TIMESTAMPTZ NOT NULL,
  sender        TEXT,
  receiver      TEXT,
  token         TEXT,
  amount        NUMERIC(78, 0), -- 基本は最小単位（lamports/mistなど）
  fee           NUMERIC(78, 0),
  method        TEXT,
  raw           JSONB,          -- 生データの保持（解析容易化）
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(chain, tx_hash, ts)
);

-- 最新残高（簡易）
CREATE TABLE IF NOT EXISTS balances (
  id            BIGSERIAL PRIMARY KEY,
  chain         chain NOT NULL,
  address       TEXT NOT NULL,
  token         TEXT NOT NULL, -- SOL/SUI or mint/type
  amount        NUMERIC(78, 0) NOT NULL,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(chain, address, token)
);

