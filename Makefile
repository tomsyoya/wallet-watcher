# --- Variables ---
FILE ?= 0001_init.sql          # デフォルトの SQL ファイル
POSTGRES_SERVICE ?= postgres   # compose のサービス名

.PHONY: up down logs migrate seed dev build

up:
	docker compose --env-file .env up -d --build

down:
	docker compose down -v

logs:
	docker compose logs -f --tail=100

# 例: make migrate                 -> /migrations/0001_init.sql を流す
#     make migrate FILE=0002_x.sql -> /migrations/0002_x.sql を流す
migrate:
	@CID=$$(docker compose ps -q $(POSTGRES_SERVICE)); \
	FILE=$${FILE:-0001_init.sql}; \
	FILE_TRIM=$$(printf '%s' "$$FILE" | awk '{$$1=$$1;print}'); \
	docker exec $$CID sh -lc 'test -f "/migrations/'"$$FILE_TRIM"'" || { echo "not found: /migrations/'"$$FILE_TRIM"'"; exit 1; }'; \
	docker exec -e PGPASSWORD=$$POSTGRES_PASSWORD $$CID \
	  sh -lc 'psql -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -f "/migrations/'"$$FILE_TRIM"'"'

psql:
	CID=$$(docker compose ps -q postgres) ; \
	docker exec -it $$CID sh -lc 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

build:
	docker compose build api

# ===========================
# テスト（build ステージで実行）
# ===========================

# build ステージをテスト用にタグ付け
build-test-image:
	docker build --target build -t wallet-watcher-test .

# ---------------------------
# Solana テスト
# ---------------------------

# モック（Solana）
test-solana: build-test-image
	@echo "==> Solana mock tests"
	docker run --rm wallet-watcher-test \
	  sh -lc 'cd /src && /usr/local/go/bin/go test ./test -v -run TestSolanaClient_Mock'

# インテグレーション（Solana）
test-integration-solana: build-test-image
	@NET=$$(docker inspect $$(docker compose ps -q postgres) --format "{{range .NetworkSettings.Networks}}{{.NetworkID}}{{end}}"); \
	echo "using network: $$NET"; \
	docker run --rm --network $$NET \
	  --env-file .env \
	  wallet-watcher-test \
	  sh -lc 'cd /src && /usr/local/go/bin/go test -tags=integration ./test -v -run TestSolana_Integration_One'

# ---------------------------
# Sui テスト
# ---------------------------

# モック（Sui）
test-sui: build-test-image
	@echo "==> Sui mock tests"
	docker run --rm wallet-watcher-test \
	  sh -lc 'cd /src && /usr/local/go/bin/go test ./test -v -run TestSuiClient_Mock'

# インテグレーション（Sui）
test-integration-sui: build-test-image
	@NET=$$(docker inspect $$(docker compose ps -q postgres) --format "{{range .NetworkSettings.Networks}}{{.NetworkID}}{{end}}"); \
	echo "using network: $$NET"; \
	docker run --rm --network $$NET \
	  --env-file .env \
	  wallet-watcher-test \
	  sh -lc 'cd /src && /usr/local/go/bin/go test -tags=integration ./test -v -run TestSui_Integration_One'