# --- Variables ---
FILE ?= 0001_init.sql          # デフォルトの SQL ファイル
POSTGRES_SERVICE ?= postgres   # compose のサービス名

.PHONY: up down logs-api logs-worker migrate seed dev build

up:
	docker compose --env-file .env up -d --build

down:
	docker compose down -v

logs-api:
	docker compose logs -f --tail=200 api

logs-worker:
	docker compose logs -f --tail=200 worker

# マイグレーション実行（/migrations/内の *.sql を順番に流す）
migrate:
	@CID=$$(docker compose ps -q postgres) ; \
	for f in $$(ls ./migrations/*.sql | sort); do \
		echo "==> running migration: $$f" ; \
		base=$$(basename $$f) ; \
		docker exec -e PGPASSWORD=$$POSTGRES_PASSWORD $$CID \
		  sh -lc 'psql -U $$POSTGRES_USER -d $$POSTGRES_DB -f /migrations/'"$$base" ; \
	done


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
	  sh -lc 'cd /src && /usr/local/go/bin/go test ./test/solana -v -run TestSolanaClient_Mock'

# インテグレーション（Solana）
test-integration-solana: build-test-image
	@NET=$$(docker inspect $$(docker compose ps -q postgres) --format "{{range .NetworkSettings.Networks}}{{.NetworkID}}{{end}}"); \
	echo "using network: $$NET"; \
	docker run --rm --network $$NET \
	  --env-file .env \
	  wallet-watcher-test \
	  sh -lc 'cd /src && /usr/local/go/bin/go test -tags=integration ./test/solana -v -run TestSolana_Integration_One'

# ---------------------------
# Sui テスト
# ---------------------------

# モック（Sui）
test-sui: build-test-image
	@echo "==> Sui mock tests"
	docker run --rm wallet-watcher-test \
	  sh -lc 'cd /src && /usr/local/go/bin/go test ./test/sui -v -run TestSuiClient_Mock'

# インテグレーション（Sui）
test-integration-sui: build-test-image
	@NET=$$(docker inspect $$(docker compose ps -q postgres) --format "{{range .NetworkSettings.Networks}}{{.NetworkID}}{{end}}"); \
	echo "using network: $$NET"; \
	docker run --rm --network $$NET \
	  --env-file .env \
	  wallet-watcher-test \
	  sh -lc 'cd /src && /usr/local/go/bin/go test -tags=integration ./test/sui -v -run TestSui_Integration_One'