-- Test queries for staging_db database (Staging Environment)

-- View all products
SELECT * FROM products;

-- View all orders with product details
SELECT 
    o.id as order_id,
    p.product_name,
    o.quantity,
    o.total_price,
    o.status,
    o.created_at
FROM orders o
JOIN products p ON o.product_id = p.id
ORDER BY o.created_at DESC;

-- Products by category
SELECT 
    category,
    COUNT(*) as product_count,
    AVG(price) as avg_price
FROM products
GROUP BY category;

-- Low stock alert
SELECT 
    product_name,
    stock,
    price
FROM products
WHERE stock < 50
ORDER BY stock;

-- Revenue by product
SELECT 
    p.product_name,
    COUNT(o.id) as order_count,
    SUM(o.total_price) as total_revenue
FROM products p
LEFT JOIN orders o ON p.id = o.product_id
GROUP BY p.id, p.product_name
ORDER BY total_revenue DESC NULLS LAST;

-- Orders by status
SELECT 
    status,
    COUNT(*) as order_count,
    SUM(total_price) as total_value
FROM orders
GROUP BY status;

-- Search products
SELECT * FROM products 
WHERE product_name ILIKE '%laptop%' OR category ILIKE '%electronics%';

-- Insert new product
INSERT INTO products (product_name, category, price, stock)
VALUES ('USB Cable', 'Electronics', 12.99, 500);

-- Insert new order
INSERT INTO orders (product_id, quantity, total_price, status)
VALUES (2, 10, 299.90, 'pending');

-- Update order status
UPDATE orders SET status = 'shipped' WHERE id = 2;

-- Inventory summary
SELECT 
    category,
    COUNT(*) as total_products,
    SUM(stock) as total_stock,
    SUM(price * stock) as inventory_value
FROM products
GROUP BY category
ORDER BY inventory_value DESC;
