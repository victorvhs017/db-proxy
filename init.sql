-- Initialize databases for testing
-- This script runs automatically when the PostgreSQL container starts

-- Create additional databases
CREATE DATABASE mydb;
CREATE DATABASE analytics_db;
CREATE DATABASE staging_db;

-- ============================================
-- DATABASE 1: postgres (default - Test Data)
-- ============================================
\c postgres

-- Create test_data table in postgres database
CREATE TABLE test_data (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    value INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample data
INSERT INTO test_data (name, value) VALUES
    ('test1', 150),
    ('test2', 200),
    ('test3', 300),
    ('test4', 400);

-- ============================================
-- DATABASE 2: mydb (User Management)
-- ============================================
\c mydb

-- Create users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create posts table
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title VARCHAR(200),
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample users
INSERT INTO users (username, email) VALUES
    ('alice', 'alice@example.com'),
    ('bob', 'bob@example.com'),
    ('charlie', 'charlie@example.com');

-- Insert sample posts
INSERT INTO posts (user_id, title, content) VALUES
    (1, 'First Post', 'Hello from Alice!'),
    (2, 'Bob''s Thoughts', 'Interesting day today.'),
    (1, 'Another Post', 'More content from Alice'),
    (3, 'Charlie Here', 'Just joined!');

-- ============================================
-- DATABASE 3: analytics_db (Analytics Data)
-- ============================================
\c analytics_db

-- Create events table
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    event_name VARCHAR(100),
    event_type VARCHAR(50),
    user_id INTEGER,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create metrics table
CREATE TABLE metrics (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(100),
    metric_value NUMERIC(10,2),
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample events
INSERT INTO events (event_name, event_type, user_id, metadata) VALUES
    ('user_login', 'authentication', 1, '{"ip": "192.168.1.1", "device": "mobile"}'),
    ('page_view', 'navigation', 1, '{"page": "/dashboard", "duration": 45}'),
    ('button_click', 'interaction', 2, '{"button": "submit", "form": "contact"}'),
    ('user_logout', 'authentication', 1, '{"session_duration": 3600}');

-- Insert sample metrics
INSERT INTO metrics (metric_name, metric_value) VALUES
    ('active_users', 1250.00),
    ('revenue', 45678.90),
    ('conversion_rate', 3.45),
    ('avg_session_time', 420.50);

-- ============================================
-- DATABASE 4: staging_db (Staging Environment)
-- ============================================
\c staging_db

-- Create products table
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    product_name VARCHAR(200),
    category VARCHAR(100),
    price NUMERIC(10,2),
    stock INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create orders table
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER,
    total_price NUMERIC(10,2),
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample products
INSERT INTO products (product_name, category, price, stock) VALUES
    ('Laptop Pro', 'Electronics', 1299.99, 50),
    ('Wireless Mouse', 'Electronics', 29.99, 200),
    ('Desk Chair', 'Furniture', 249.99, 30),
    ('Monitor 27"', 'Electronics', 399.99, 75),
    ('Keyboard RGB', 'Electronics', 89.99, 150);

-- Insert sample orders
INSERT INTO orders (product_id, quantity, total_price, status) VALUES
    (1, 2, 2599.98, 'completed'),
    (2, 5, 149.95, 'pending'),
    (3, 1, 249.99, 'shipped'),
    (4, 3, 1199.97, 'completed');

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE postgres TO postgres;
GRANT ALL PRIVILEGES ON DATABASE mydb TO postgres;
GRANT ALL PRIVILEGES ON DATABASE analytics_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE staging_db TO postgres;