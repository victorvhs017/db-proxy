# Database Proxy Makefile

.PHONY: help db-up db-down db-logs db-connect db-status proxy build clean test replay

help: ## Show this help message
	@echo "📚 Database Proxy - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""

db-up: ## Start PostgreSQL database in Docker
	@echo "🚀 Starting PostgreSQL database..."
	@docker-compose up -d
	@echo "⏳ Waiting for database to be ready..."
	@sleep 3
	@docker-compose exec -T postgres pg_isready -U postgres || (echo "⚠️  Database not ready yet, waiting..."; sleep 3)
	@echo "✅ Database is ready!"

db-init: ## Initialize 4 test databases with sample data
	@echo "📊 Initializing databases..."
	@echo "Creating: postgres, mydb, analytics_db, staging_db"
	@docker-compose exec -T postgres psql -U postgres < init.sql
	@echo "✅ All 4 databases initialized with test data!"

db-down: ## Stop PostgreSQL database
	@echo "🛑 Stopping PostgreSQL database..."
	@docker-compose down

db-logs: ## Show database logs
	@docker-compose logs -f postgres

db-connect: ## Connect directly to database (bypass proxy)
	@echo "🔌 Connecting to database directly..."
	@docker-compose exec postgres psql -U postgres -d postgres

db-status: ## Check database status
	@echo "📊 Database Status:"
	@docker-compose ps
	@echo ""
	@docker-compose exec -T postgres pg_isready -U postgres && echo "✅ Database is running" || echo "❌ Database is not running"

proxy: ## Run the proxy
	@echo "🚀 Starting Database Proxy..."
	@go run main.go

build: ## Build the proxy binary
	@echo "🔨 Building proxy binary..."
	@go build -o dbproxy
	@echo "✅ Binary built: ./dbproxy"

clean: ## Clean up Docker volumes and binary
	@echo "🧹 Cleaning up..."
	@docker-compose down -v
	@rm -f dbproxy
	@echo "✅ Cleanup complete"

test: ## Start database and show test instructions
	@$(MAKE) db-up
	@echo ""
	@echo "📝 Databases ready for testing!"
	@echo ""
	@echo "Available databases: postgres, mydb, analytics_db, staging_db"
	@echo ""
	@echo "In another terminal, run:"
	@echo "  make proxy"
	@echo ""
	@echo "The proxy will let you select which database to monitor."
	@echo "It will generate unique session credentials for you."
	@echo ""
	@echo "Test query files available in test-queries/ directory"

test-queries: ## List available test query files
	@echo "📝 Test Query Files:"
	@echo ""
	@echo "  test-queries/postgres.sql      - Basic test data"
	@echo "  test-queries/mydb.sql          - Users & posts"
	@echo "  test-queries/analytics_db.sql  - Events & metrics"
	@echo "  test-queries/staging_db.sql    - Products & orders"
	@echo ""
	@echo "💡 Usage: psql -h localhost -p 5433 -U <session_user> -d <db> -f test-queries/<file>.sql"

replay: ## Replay a session interactively (or provide FILE=path)
	@if [ -z "$(FILE)" ]; then \
		go run cmd/replay/main.go; \
	else \
		go run cmd/replay/main.go $(FILE); \
	fi