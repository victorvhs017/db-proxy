-- Test queries for analytics_db database (Analytics Data)

-- View all events
SELECT * FROM events ORDER BY created_at DESC;

-- View all metrics
SELECT * FROM metrics ORDER BY recorded_at DESC;

-- Events by type
SELECT 
    event_type,
    COUNT(*) as event_count
FROM events
GROUP BY event_type
ORDER BY event_count DESC;

-- Events with specific metadata (JSONB queries)
SELECT 
    event_name,
    event_type,
    metadata->>'ip' as ip_address,
    metadata->>'device' as device,
    created_at
FROM events
WHERE metadata ? 'ip';

-- Filter by user
SELECT * FROM events WHERE user_id = 1;

-- Latest metrics snapshot
SELECT 
    metric_name,
    metric_value,
    recorded_at
FROM metrics
ORDER BY recorded_at DESC
LIMIT 10;

-- Average metric values
SELECT 
    metric_name,
    AVG(metric_value) as avg_value
FROM metrics
GROUP BY metric_name;

-- Insert new event
INSERT INTO events (event_name, event_type, user_id, metadata)
VALUES ('api_call', 'system', 5, '{"endpoint": "/api/users", "status": 200}');

-- Insert new metric
INSERT INTO metrics (metric_name, metric_value)
VALUES ('response_time', 125.50);

-- Event timeline (last hour)
SELECT 
    event_name,
    event_type,
    user_id,
    created_at
FROM events
WHERE created_at > NOW() - INTERVAL '1 hour'
ORDER BY created_at;
