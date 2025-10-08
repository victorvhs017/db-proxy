-- Sample test queries to run through the proxy
-- Connect using: psql -h localhost -p 5433 -U pseudo_user -d postgres

-- Test 1: Simple SELECT
SELECT * FROM test_data;

-- Test 2: INSERT
INSERT INTO test_data (name, value) VALUES ('test4', 400);

-- Test 3: UPDATE
UPDATE test_data SET value = 150 WHERE name = 'test1';

-- Test 4: DELETE
DELETE FROM test_data WHERE name = 'test3';

-- Test 5: Complex JOIN (switch to mydb database first: \c mydb)
-- psql -h localhost -p 5433 -U dev_user -d mydb
SELECT 
    u.username,
    u.email,
    COUNT(p.id) as post_count
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
GROUP BY u.id, u.username, u.email
ORDER BY post_count DESC;

-- Test 6: CREATE TABLE
CREATE TABLE temp_test (
    id SERIAL PRIMARY KEY,
    description TEXT
);

-- Test 7: DROP TABLE
DROP TABLE IF EXISTS temp_test;

-- Test 8: Transaction
BEGIN;
INSERT INTO test_data (name, value) VALUES ('transaction_test', 999);
SELECT * FROM test_data WHERE name = 'transaction_test';
ROLLBACK;

-- Test 9: Error handling (intentional error)
SELECT * FROM non_existent_table;

-- Test 10: Prepared statement-like query
SELECT * FROM test_data WHERE value > 100 AND value < 300;

