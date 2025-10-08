# Example Workflow - Complete Session Recording and Replay

This document demonstrates a complete workflow of recording a database session and replaying it.

## Step 1: Start the Database and Proxy

```bash
# Terminal 1: Start the database
make db-up

# Terminal 2: Start the proxy
make proxy
```

You should see:
```
✅ Proxy Server Started!

📋 Available Pseudo Credentials:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Pseudo User: pseudo_user
  Password: (any password works)
  Database: postgres
  → Maps to: postgres://postgres:***@localhost:5432/postgres

🔌 Connect using:
  psql -h localhost -p 5433 -U pseudo_user -d postgres

🔍 Monitoring all queries and responses...
```

## Step 2: Connect and Execute Queries

```bash
# Terminal 3: Connect through the proxy
psql -h localhost -p 5433 -U pseudo_user -d postgres
```

Execute some queries:
```sql
-- Query 1: Select all data
SELECT * FROM test_data;

-- Query 2: Insert new record
INSERT INTO test_data (name, value) VALUES ('new_record', 999);

-- Query 3: Update a record
UPDATE test_data SET value = 1000 WHERE name = 'new_record';

-- Query 4: Select with filter
SELECT * FROM test_data WHERE value > 500;

-- Query 5: Delete a record
DELETE FROM test_data WHERE name = 'new_record';

-- Exit
\q
```

## Step 3: View Proxy Output

In Terminal 2 (proxy), you'll see real-time output:

```
[session_xxx] 🔗 New connection from 127.0.0.1:xxxxx
[session_xxx] 👤 Pseudo user attempting connection: pseudo_user (database: postgres)
[session_xxx] ✅ Credentials mapped: pseudo_user → postgres
[session_xxx] 🔌 Connected to real database
[session_xxx] 📝 Session recording started

[session_xxx] 📝 Query #1 @ 21:30:15.123
[session_xxx] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[session_xxx] SELECT * FROM test_data;
[session_xxx] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[session_xxx]    📊 Results:
[session_xxx]    ┌──────┬───────┬───────┬────────────────────────────┐
[session_xxx]    │ id   │ name  │ value │ created_at                 │
[session_xxx]    ├──────┼───────┼───────┼────────────────────────────┤
[session_xxx]    │ 1    │ test1 │ 150   │ 2025-10-07 22:42:12.977496 │
[session_xxx]    │ 2    │ test2 │ 200   │ 2025-10-07 22:42:12.977496 │
[session_xxx]    │ 3    │ test3 │ 300   │ 2025-10-07 22:42:12.977496 │
[session_xxx]    └──────┴───────┴───────┴────────────────────────────┘
[session_xxx]    3 row(s)

[session_xxx]    ✓ SELECT 3

[session_xxx] 📝 Query #2 @ 21:30:22.456
[session_xxx] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[session_xxx] INSERT INTO test_data (name, value) VALUES ('new_record', 999);
[session_xxx] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[session_xxx]    → No result set
[session_xxx]    ✓ INSERT 0 1

... more queries ...

[session_xxx] 🔌 Connection closed
```

## Step 4: Check the Session File

```bash
# List recorded sessions
ls -la sessions/

# View the session file
cat sessions/pseudo_user_2025-10-07_21-30-15.json
```

Session file structure:
```json
[
  {
    "type": "session_activity",
    "timestamp": "2025-10-07T21:30:15Z",
    "data": {
      "activity": "session_start",
      "details": {
        "session_id": "session_1759881348861829000",
        "pseudo_user": "pseudo_user",
        "start_time": "2025-10-07T21:30:15Z"
      }
    }
  },
  {
    "type": "session_activity",
    "timestamp": "2025-10-07T21:30:15Z",
    "data": {
      "activity": "connection",
      "details": {
        "pseudo_user": "pseudo_user",
        "real_user": "postgres",
        "database": "postgres",
        "real_db_host": "localhost",
        "real_db_port": "5432"
      }
    }
  },
  {
    "type": "query",
    "timestamp": "2025-10-07T21:30:15Z",
    "data": {
      "query_number": 1,
      "sql": "SELECT * FROM test_data;"
    }
  },
  {
    "type": "query_result",
    "timestamp": "2025-10-07T21:30:15Z",
    "data": {
      "query_number": 1,
      "columns": ["id", "name", "value", "created_at"],
      "rows": [
        ["1", "test1", "150", "2025-10-07 22:42:12.977496"],
        ["2", "test2", "200", "2025-10-07 22:42:12.977496"],
        ["3", "test3", "300", "2025-10-07 22:42:12.977496"]
      ],
      "row_count": 3,
      "command_tag": "SELECT 3"
    }
  },
  ...
]
```

## Step 5: Replay the Session

```bash
# Replay the session
make replay FILE=sessions/pseudo_user_2025-10-07_21-30-15.json
```

Output:
```
╔═══════════════════════════════════════════════════╗
║          Session Replay Tool                      ║
║          Re-execute recorded queries              ║
╚═══════════════════════════════════════════════════╝

✅ Loaded 15 events from sessions/pseudo_user_2025-10-07_21-30-15.json
✅ Connected to database

📌 [Event 1] Session Activity: session_start
📌 [Event 2] Session Activity: connection

📝 [Event 3] Query #1 @ 2025-10-07T21:30:15Z
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SELECT * FROM test_data;
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ✓ Query executed: 3 rows returned
   ✓ Results match recorded data (3 rows)

📝 [Event 5] Query #2 @ 2025-10-07T21:30:22Z
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
INSERT INTO test_data (name, value) VALUES ('new_record', 999);
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ✓ Query executed: 0 rows returned

... more queries ...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Replay Summary:
   Queries executed: 5
   Errors: 0
   Results matched: 2
   Results mismatched: 0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ Session replay completed successfully!
```

## Step 6: Using with Different Databases

You can replay sessions against different database instances:

```bash
# Replay against production (with read-only user for safety!)
go run cmd/replay/main.go sessions/pseudo_user_2025-10-07_21-30-15.json \
  --db-url "postgres://readonly_user:pass@prod-db.example.com:5432/mydb"

# Replay against staging
go run cmd/replay/main.go sessions/pseudo_user_2025-10-07_21-30-15.json \
  --db-url "postgres://user:pass@staging-db.example.com:5432/mydb"
```

## Use Cases

### 1. Regression Testing
Record a session with known good data, then replay after schema changes:
```bash
# Before migration
make proxy  # Record session
# ... execute tests ...

# After migration
make replay FILE=sessions/test_session.json  # Verify results match
```

### 2. Security Audit
Review what a user did during their session:
```bash
# 1. User connects and works
# 2. Session is recorded automatically
# 3. Review the JSON file
cat sessions/suspicious_user_2025-10-07_21-30-15.json | jq '.[] | select(.type=="query")'

# 4. Replay to verify behavior
make replay FILE=sessions/suspicious_user_2025-10-07_21-30-15.json
```

### 3. Debugging
Reproduce a bug by replaying the exact sequence of queries:
```bash
# User reports issue
# Get their session file
make replay FILE=sessions/user_with_issue.json
# Debug while watching queries execute
```

### 4. Performance Testing
Compare query performance between environments:
```bash
# Record on local
make proxy  # Session recorded

# Replay on staging and measure
time make replay FILE=sessions/performance_test.json

# Replay on production (read-only)
time go run cmd/replay/main.go sessions/performance_test.json --db-url $PROD_URL
```

## Advanced: Parsing Session Files with jq

Extract specific information from session files:

```bash
# Get all queries
cat sessions/pseudo_user_*.json | jq '.[] | select(.type=="query") | .data.sql'

# Count queries by type
cat sessions/pseudo_user_*.json | jq '.[] | select(.type=="query") | .data.sql' | grep -oE '^[A-Z]+' | sort | uniq -c

# Find slow queries (if you add timing)
cat sessions/pseudo_user_*.json | jq '.[] | select(.type=="query_result" and .data.row_count > 1000)'

# Get all errors
cat sessions/pseudo_user_*.json | jq '.[] | select(.type=="query_result" and .data.error != null)'
```

## Cleanup

```bash
# Remove old session files
rm sessions/pseudo_user_2025-10-07_*.json

# Or remove all
rm -rf sessions/

# Stop database
make db-down
```
