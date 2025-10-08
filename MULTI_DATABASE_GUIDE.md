# Multi-Database Setup Guide

This guide explains how to use the proxy's multi-database feature with random session credentials.

## Overview

The proxy now supports **4 pre-configured databases** that you can choose from when starting the proxy. Each session gets **unique random credentials** that expire when the proxy stops.

## Available Databases

### 1. **Production DB** (`postgres`)
Main production database with test data.

**Tables:**
- `test_data` - Simple test records with id, name, value, created_at

**Use Cases:**
- Basic CRUD operations
- Testing aggregate functions
- Simple SELECT/INSERT/UPDATE queries

**Test Queries:** `test-queries/postgres.sql`

---

### 2. **User Management** (`mydb`)
User and content management system.

**Tables:**
- `users` - User accounts (id, username, email, created_at)
- `posts` - User posts (id, user_id, title, content, created_at)

**Use Cases:**
- JOIN operations
- User activity tracking
- Content management queries

**Test Queries:** `test-queries/mydb.sql`

---

### 3. **Analytics** (`analytics_db`)
Analytics and event tracking database.

**Tables:**
- `events` - Event logs (id, event_name, event_type, user_id, metadata JSONB, created_at)
- `metrics` - Performance metrics (id, metric_name, metric_value, recorded_at)

**Use Cases:**
- JSONB queries
- Event tracking
- Analytics aggregations
- Time-series data

**Test Queries:** `test-queries/analytics_db.sql`

---

### 4. **Staging** (`staging_db`)
E-commerce staging environment.

**Tables:**
- `products` - Product catalog (id, product_name, category, price, stock, created_at)
- `orders` - Order records (id, product_id, quantity, total_price, status, created_at)

**Use Cases:**
- E-commerce queries
- Inventory management
- Revenue calculations
- Order tracking

**Test Queries:** `test-queries/staging_db.sql`

---

## Quick Start

### Step 1: Initialize Databases

```bash
# Start PostgreSQL container
make db-up

# Initialize all 4 databases with test data
make db-init
```

This creates and populates all 4 databases.

### Step 2: Start the Proxy

```bash
make proxy
```

You'll see an interactive menu:

```
╔═══════════════════════════════════════════════════╗
║         Database Proxy - Session Recorder         ║
║              Intercept & Monitor DB               ║
╚═══════════════════════════════════════════════════╝

🗄️  Select Database to Monitor:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  │ 1) Production DB
  │    Main production database (postgres)
  │    Database: postgres
  ├──────────────────────────────────────────────────
  │ 2) User Management
  │    User and posts database (mydb)
  │    Database: mydb
  ├──────────────────────────────────────────────────
  │ 3) Analytics
  │    Analytics and events database (analytics_db)
  │    Database: analytics_db
  ├──────────────────────────────────────────────────
  │ 4) Staging
  │    Staging environment (staging_db)
  │    Database: staging_db
  └──────────────────────────────────────────────────

Select a database [1-4]: 
```

### Step 3: Select Database

Type a number (1-4) and press Enter. For example, type `2` to select User Management.

### Step 4: Get Your Session Credentials

After selecting, you'll see **unique session credentials**:

```
✅ Selected: User Management
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 Session Credentials (Valid for this session only):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Database: User Management
  Session User: session_a3f2c8d1
  Password: (any password works)
  Target DB: mydb
  → Maps to: postgres://postgres:***@localhost:5432/mydb

🔌 Connect using:
  psql -h localhost -p 5433 -U session_a3f2c8d1 -d mydb

Or with connection string:
  postgres://session_a3f2c8d1:anypassword@localhost:5433/mydb

🔍 Monitoring all queries and responses...

💡 Tip: Session credentials change each time you run the proxy
```

**Important:** 
- The `session_a3f2c8d1` username is **random** and changes each time
- These credentials are **only valid for this proxy session**
- When you restart the proxy, you'll get new credentials

### Step 5: Connect and Query

Open a new terminal and connect using the displayed credentials:

```bash
# Using the session user shown above
psql -h localhost -p 5433 -U session_a3f2c8d1 -d mydb
```

Or run test queries:

```bash
psql -h localhost -p 5433 -U session_a3f2c8d1 -d mydb -f test-queries/mydb.sql
```

## Understanding Session Credentials

### Why Random Credentials?

1. **Security**: Each session gets unique credentials that can't be reused
2. **Tracking**: Easy to identify which session executed which queries
3. **Isolation**: Multiple proxy instances can run simultaneously without conflicts
4. **Audit**: Session files are named with the session user and timestamp

### Credential Format

Session usernames follow the pattern:
```
session_<8_hex_chars>
```

Example: `session_a3f2c8d1`, `session_7b9c4e2f`

### Session Files

When you connect and run queries, a JSON file is created:

```
sessions/session_a3f2c8d1_2025-10-08_14-30-45.json
```

This file contains all queries and results from that session.

## Testing Workflow Examples

### Example 1: Test User Management

```bash
# Terminal 1: Start proxy and select User Management (option 2)
make proxy

# Terminal 2: Connect with displayed credentials
psql -h localhost -p 5433 -U session_XXXXXXXX -d mydb

# Run some queries
SELECT * FROM users;
SELECT * FROM posts JOIN users ON posts.user_id = users.id;
```

### Example 2: Test Analytics JSONB Queries

```bash
# Terminal 1: Start proxy and select Analytics (option 3)
make proxy

# Terminal 2: Run analytics test queries
psql -h localhost -p 5433 -U session_XXXXXXXX -d analytics_db -f test-queries/analytics_db.sql
```

### Example 3: Compare Multiple Sessions

```bash
# Session 1: Test against User Management
make proxy  # Select option 2
# ... run queries ...
# Ctrl+C to stop

# Session 2: Test against Staging
make proxy  # Select option 4
# ... run queries ...
# Ctrl+C to stop

# Now replay and compare both sessions
make replay  # Select session to replay
```

## Replaying Sessions

After running queries, you can replay any session:

```bash
make replay
```

**Step 1: Select Session**

The tool shows all recorded sessions:

```
╔═══════════════════════════════════════════════════╗
║          Session Replay Tool                      ║
║          Re-execute recorded queries              ║
╚═══════════════════════════════════════════════════╝

📂 Available Session Files:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  │ 1) session_a3f2c8d1_2025-10-08_14-30-45.json
  │    User: session_a3f2c8d1
  │    Date: 2025-10-08 14:30:45
  │    Size: 2.4 KB
  ├──────────────────────────────────────────────────
  │ 2) session_7b9c4e2f_2025-10-08_13-15-20.json
  │    User: session_7b9c4e2f
  │    Date: 2025-10-08 13:15:20
  │    Size: 3.1 KB
  └──────────────────────────────────────────────────

Select a session to replay [1-2] or 'q' to quit: 1
```

**Step 2: Enter Database URL**

You'll be prompted to specify which database to replay against:

```
🗄️  Database Connection
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Examples:
  postgres://postgres:postgres@localhost:5432/postgres
  postgres://postgres:postgres@localhost:5432/mydb
  postgres://postgres:postgres@localhost:5432/analytics_db
  postgres://postgres:postgres@localhost:5432/staging_db

Enter database URL: postgres://postgres:postgres@localhost:5432/mydb
```

**Why specify the database?**
- You can replay sessions against different databases for testing
- Useful for validating migrations or schema changes
- Allows testing same queries across production/staging/dev environments

## Tips & Best Practices

### 1. Document Your Tests
Keep track of which database you're testing and why:
- Use session files to review test results later
- Name your test scenarios in comments

### 2. Multiple Concurrent Sessions
You can run multiple proxy instances on different ports:
```bash
# Edit main.go to use different port, or
# Connect multiple clients to the same proxy session
```

### 3. Clean Up Old Sessions
```bash
# Remove old session files
rm sessions/*.json

# Or keep for audit trail
```

### 4. Database Management
```bash
# Reset databases to initial state
make db-down
make db-up
make db-init

# View database directly (bypass proxy)
make db-connect
```

## Troubleshooting

### Can't Connect with Session Credentials

**Problem:** Connection refused or authentication failed

**Solution:**
1. Make sure the proxy is running
2. Use the **exact** username shown in the proxy output
3. Copy-paste the connection string to avoid typos
4. Try any password - it's ignored

### Database Not Found

**Problem:** Database "mydb" does not exist

**Solution:**
```bash
# Reinitialize databases
make db-init
```

### Wrong Credentials After Restart

**Problem:** Old session credentials don't work

**Solution:**
- This is expected! Session credentials change on each proxy start
- Always use the credentials displayed in the current proxy session

## Next Steps

1. Explore each database with the provided test queries
2. Try writing your own complex queries
3. Record sessions and replay them for validation
4. Integrate into your testing workflow

For more information, see:
- [README.md](README.md) - Main documentation
- [EXAMPLE_WORKFLOW.md](EXAMPLE_WORKFLOW.md) - Complete workflow example
- Test query files in `test-queries/` directory
