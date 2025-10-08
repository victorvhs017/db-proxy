-- Test queries for mydb database (User Management)

-- View all users
SELECT * FROM users;

-- View all posts with user info
SELECT 
    p.id,
    p.title,
    p.content,
    u.username,
    p.created_at
FROM posts p
JOIN users u ON p.user_id = u.id
ORDER BY p.created_at DESC;

-- Count posts per user
SELECT 
    u.username,
    COUNT(p.id) as post_count
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
GROUP BY u.username
ORDER BY post_count DESC;

-- Find user by email
SELECT * FROM users WHERE email LIKE '%alice%';

-- Recent posts
SELECT * FROM posts WHERE created_at > NOW() - INTERVAL '1 day';

-- Insert new user
INSERT INTO users (username, email) VALUES ('dave', 'dave@example.com');

-- Insert new post
INSERT INTO posts (user_id, title, content) 
VALUES (1, 'New Post', 'Testing the proxy system');

-- User activity summary
SELECT 
    u.username,
    u.email,
    COUNT(p.id) as total_posts,
    MAX(p.created_at) as last_post_date
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
GROUP BY u.id, u.username, u.email;
