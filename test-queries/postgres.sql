-- Test queries for postgres database (Test Data)

-- View all test data
SELECT * FROM test_data;

-- Filter by value
SELECT * FROM test_data WHERE value > 200;

-- Order by name
SELECT * FROM test_data ORDER BY name;

-- Aggregate queries
SELECT COUNT(*) as total_records FROM test_data;
SELECT AVG(value) as avg_value FROM test_data;
SELECT MAX(value) as max_value, MIN(value) as min_value FROM test_data;

-- Insert new record
INSERT INTO test_data (name, value) VALUES ('test5', 500);

-- Update record
UPDATE test_data SET value = 250 WHERE name = 'test2';

-- Check recent inserts
SELECT * FROM test_data WHERE created_at > NOW() - INTERVAL '1 hour';
