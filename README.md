# Database Proxy - Session Recorder

A transparent PostgreSQL proxy that intercepts database connections, swaps pseudo credentials for real ones, and logs all queries and responses in real-time. All session activity is automatically recorded to JSON files for audit and replay.

## Features

- 🗄️ **Multi-Database Support**: Choose from 4 pre-configured test databases
- 🔐 **Credential Swapping**: Session-specific credentials automatically swapped for real ones
- 🎲 **Random Session Credentials**: Each proxy session gets unique temporary credentials
- 📝 **Query Logging**: Log every SQL query executed during the session
- 📊 **Response Monitoring**: Track query results and database responses
- 💾 **Session Recording**: Automatically save all queries, results, and activities to JSON files
- 🔄 **Interactive Session Replay**: Select and replay any recorded session
- 🎨 **Beautiful CLI**: Color-coded output for easy monitoring
- ⚡ **Low Latency**: Built with Go for minimal overhead
- 🔍 **Protocol Support**: Works with both Simple Query Protocol (psql) and Extended Query Protocol (DBeaver, pgAdmin, etc.)

## How It Works

```
User App → Pseudo Credentials → Proxy (localhost:5433) → Real Credentials → Real Database
                                   ↓
                        Query Logger & JSON Recorder
                                   ↓
                        sessions/username_timestamp.json
```

1. User connects to proxy using pseudo credentials
2. Proxy intercepts the connection and authenticates
3. Proxy swaps credentials and connects to real database
4. Session recorder creates a JSON file in `sessions/` directory
5. All queries and responses flow through proxy and get logged to console and JSON
6. User application works normally, unaware of the interception
7. When session ends, JSON file is finalized with complete session history

## Prerequisites

- Go 1.21 or higher
- PostgreSQL database (for testing)

## Installation

```bash
# Clone the repository
cd /Users/victordossantos/Documents/infisical/session-recorder

# Download dependencies
go mod download

# Build the application
go build -o dbproxy

# Run the proxy
./dbproxy
```

## Quick Start

> 📖 **Helpful Guides:**
> - [MULTI_DATABASE_GUIDE.md](MULTI_DATABASE_GUIDE.md) - Complete guide for multi-database feature
> - [EXAMPLE_WORKFLOW.md](EXAMPLE_WORKFLOW.md) - End-to-end example with replay

### 1. Start the Proxy

```bash
make proxy
```

**Interactive Database Selection:**
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

Select a database [1-4]: 2

✅ Selected: User Management

📋 Session Credentials (Valid for this session only):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Database: User Management
  Session User: session_a3f2c8d1
  Password: (any password works)
  Target DB: mydb
  → Maps to: postgres://postgres:***@localhost:5432/mydb

🔌 Connect using:
  psql -h localhost -p 5433 -U session_a3f2c8d1 -d mydb

🔍 Monitoring all queries and responses...

💡 Tip: Session credentials change each time you run the proxy
```

### 2. Connect Through the Proxy

In another terminal, connect using the **displayed session credentials**:

```bash
# Use the exact username and database shown by the proxy
psql -h localhost -p 5433 -U session_a3f2c8d1 -d mydb

# Or use connection string
psql postgres://session_a3f2c8d1:anypassword@localhost:5433/mydb
```

When prompted for password, enter anything (it will be ignored).

**💡 Note**: Each time you restart the proxy, you'll get new session credentials!

### 3. Run Queries

Execute SQL queries as normal:

```sql
SELECT * FROM users;
CREATE TABLE test (id serial, name text);
INSERT INTO test (name) VALUES ('Alice');
```

### 4. Watch the Logs

The proxy will log everything in real-time:

```
[session_xxx] 🔗 New connection from 127.0.0.1:xxxxx
[session_xxx] 👤 Pseudo user attempting connection: pseudo_user (database: postgres)
[session_xxx] ✅ Credentials mapped: pseudo_user → postgres
[session_xxx] 🔌 Connected to real database
[session_xxx] 🎉 Authentication successful! Starting to monitor queries...

[session_xxx] 📝 Query #1 @ 14:30:45.123
   SELECT * FROM users WHERE id = 1;
[session_xxx]    ✓ SELECT 1

[session_xxx] 📝 Query #2 @ 14:30:52.456
   INSERT INTO test (name) VALUES ('Alice');
[session_xxx]    ✓ INSERT 0 1
```

## Configuration

Edit the credential map in `main.go`:

```go
CredentialMap: map[string]proxy.Credentials{
    "pseudo_user": {
        Username: "postgres",      // Real database username
        Password: "postgres",      // Real database password
        Database: "postgres",      // Real database name
    },
    "dev_user": {
        Username: "admin",
        Password: "secret123",
        Database: "mydb",
    },
},
```

You can also change the proxy and database ports:

```go
ProxyHost:      "localhost",  // Proxy listen address
ProxyPort:      "5433",       // Proxy listen port
RealDBHost:     "localhost",  // Real database host
RealDBPort:     "5432",       // Real database port
```

## Architecture

```
.
├── main.go                 # CLI entry point and configuration
├── cmd/
│   └── replay/
│       └── main.go        # Session replay tool
├── proxy/
│   ├── config.go          # Configuration structures
│   ├── proxy.go           # Main proxy server and session management
│   ├── postgresql.go      # PostgreSQL wire protocol handler
│   └── recorder.go        # JSON session recorder
├── sessions/              # Recorded session JSON files
├── go.mod                 # Go dependencies
├── Makefile              # Build and run commands
└── README.md             # This file
```

### Key Components

- **Main CLI** (`main.go`): Starts the proxy, displays connection info, handles graceful shutdown
- **Proxy Server** (`proxy.go`): TCP listener, session management, connection handling
- **PostgreSQL Handler** (`postgresql.go`): Wire protocol parsing, credential swapping, query/response logging
- **Session Recorder** (`recorder.go`): Records all queries, results, and activities to JSON files
- **Replay Tool** (`cmd/replay/main.go`): Re-executes recorded sessions and validates results

## PostgreSQL Wire Protocol

The proxy implements the PostgreSQL wire protocol at a binary level:

1. **Startup Phase**: Intercepts and modifies startup message with real credentials
2. **Authentication**: Proxies authentication messages between client and server
3. **Query Phase**: Parses query messages ('Q'), logs SQL, forwards to database
4. **Response Phase**: Logs responses (CommandComplete, ErrorResponse, DataRow)

## Testing

### Quick Start with Make

The easiest way to get started:

```bash
# Start the PostgreSQL database
make db-up

# In another terminal, start the proxy
make proxy

# In a third terminal, connect through the proxy
psql -h localhost -p 5433 -U pseudo_user -d postgres
```

### Available Make Commands

```bash
make help          # Show all available commands
make db-up         # Start PostgreSQL database in Docker
make db-down       # Stop PostgreSQL database
make db-logs       # Show database logs
make db-connect    # Connect directly to database (bypass proxy)
make db-status     # Check database status
make proxy         # Run the proxy
make build         # Build the proxy binary
make clean         # Clean up Docker volumes and binary
make test          # Start database and show test instructions
make replay FILE=  # Replay a recorded session file
```

### Manual Setup (without Make)

#### Set up a local PostgreSQL database:

```bash
# Using Docker Compose (recommended)
docker-compose up -d

# Wait for database to be ready
docker-compose exec postgres pg_isready -U postgres

# Or using Docker directly
docker run --name test-postgres -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres

# Or use an existing PostgreSQL installation
```

#### Run the proxy:

```bash
go run main.go
```

#### Test connection:

```bash
# Using psql with pseudo credentials
psql -h localhost -p 5433 -U pseudo_user -d postgres

# Or connect to mydb database
psql -h localhost -p 5433 -U dev_user -d mydb

# Using any PostgreSQL client
# Connection string: postgresql://pseudo_user:any@localhost:5433/postgres
```

### Test Database Schema

The test database includes:

**postgres database:**
- `test_data` table with sample records

**mydb database:**
- `users` table with 3 sample users
- `posts` table with 4 sample posts

Use the provided `test-queries.sql` file for sample queries to test the proxy.

## Session Recording

Every database session is automatically recorded to a JSON file in the `sessions/` directory.

### File Format

Files are named: `{username}_{timestamp}.json`

Example: `pseudo_user_2025-10-07_20-55-54.json`

### JSON Structure

Each session file contains an array of events with three types:

#### 1. Session Activity Events
```json
{
  "type": "session_activity",
  "timestamp": "2025-10-07T20:55:54Z",
  "data": {
    "activity": "session_start",
    "details": {
      "session_id": "session_1759881348861829000",
      "pseudo_user": "pseudo_user",
      "start_time": "2025-10-07T20:55:54Z"
    }
  }
}
```

#### 2. Query Events
```json
{
  "type": "query",
  "timestamp": "2025-10-07T20:56:01Z",
  "data": {
    "query_number": 1,
    "sql": "SELECT * FROM test_data;"
  }
}
```

#### 3. Query Result Events
```json
{
  "type": "query_result",
  "timestamp": "2025-10-07T20:56:01Z",
  "data": {
    "query_number": 1,
    "columns": ["id", "name", "value", "created_at"],
    "rows": [
      ["1", "test1", "150", "2025-10-07 22:42:12.977496"],
      ["2", "test2", "200", "2025-10-07 22:42:12.977496"]
    ],
    "row_count": 2,
    "command_tag": "SELECT 2"
  }
}
```

### Replaying Sessions

The replay tool features an **interactive menu** to select and replay sessions:

```bash
# Interactive mode - shows menu of available sessions
make replay

# Or specify a file directly
make replay FILE=sessions/pseudo_user_2025-10-07_20-55-54.json

# Or provide database URL via command line (skips prompt)
go run cmd/replay/main.go sessions/pseudo_user_2025-10-07_20-55-54.json --db-url postgres://user:pass@host:5432/dbname
```

**Interactive Flow:**

**1. Select Session:**
```
╔═══════════════════════════════════════════════════╗
║          Session Replay Tool                      ║
║          Re-execute recorded queries              ║
╚═══════════════════════════════════════════════════╝

📂 Available Session Files:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  │ 1) session_a3f2c8d1_2025-10-08_21-53-09.json
  │    User: session_a3f2c8d1
  │    Date: 2025-10-08 21:53:09
  │    Size: 2.4 KB
  ├──────────────────────────────────────────────────
  │ 2) session_7b9c4e2f_2025-10-08_20-15-30.json
  │    User: session_7b9c4e2f
  │    Date: 2025-10-08 20:15:30
  │    Size: 1.8 KB
  └──────────────────────────────────────────────────

Select a session to replay [1-2] or 'q' to quit: 1
```

**2. Enter Database URL:**
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

The replay tool will:
- ✅ Execute each query in order
- ✅ Compare results with recorded data
- ✅ Show any differences
- ✅ Provide a summary of matches/mismatches

Example output:
```
📝 [Event 5] Query #1 @ 2025-10-07T20:56:01Z
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SELECT * FROM test_data;
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ✓ Query executed: 5 rows returned
   ✓ Results match recorded data (5 rows)

📊 Replay Summary:
   Queries executed: 12
   Errors: 0
   Results matched: 12
   Results mismatched: 0
```

### Use Cases

- **Security Auditing**: Monitor all database operations by users with complete audit trail
- **Credential Management**: Provide temporary/pseudo credentials that map to real ones
- **Query Analytics**: Analyze query patterns and performance from JSON logs
- **Session Replay**: Recreate exactly what happened during a session and verify results
- **Regression Testing**: Use recorded sessions as test cases for database changes
- **Compliance**: Keep detailed records of all database access and modifications
- **Development**: Debug database interactions without modifying application code
- **Forensics**: Investigate security incidents with complete session history
- **Data Validation**: Verify that database changes produce expected results

## Limitations (Current POC)

- Only supports PostgreSQL (MySQL support can be added)
- Password authentication only (no SSL/TLS yet)
- Simple credential mapping (no dynamic auth)
- Results limited to 20 rows in console display (full results saved to JSON)

## Future Enhancements

- [ ] SSL/TLS support
- [ ] MySQL protocol support
- [ ] Full result set logging
- [ ] Query performance metrics
- [ ] Session replay functionality
- [ ] API for dynamic credential management
- [ ] Web UI for monitoring
- [ ] Export logs to various formats (JSON, CSV)

## License

MIT License

## Contributing

Contributions welcome! This is a POC project for database session recording and credential management.

