-- name: GetDashboardStats :one
SELECT
    (SELECT COUNT(*) FROM products WHERE deleted_at IS NULL) AS total_products,
    (SELECT COUNT(*) FROM brands WHERE deleted_at IS NULL) AS total_brands,
    (SELECT COUNT(*) FROM categories WHERE deleted_at IS NULL) AS total_categories,
    (SELECT COUNT(*) FROM users WHERE role = 'CUSTOMER') AS total_customers,
    (SELECT COUNT(*) FROM orders WHERE deleted_at IS NULL) AS total_orders,
    (SELECT COALESCE(SUM(total_price::numeric), 0) FROM orders WHERE status = 'COMPLETED' AND deleted_at IS NULL) AS total_revenue;

-- name: ListRecentOrders :many
SELECT 
    o.id,
    o.order_number,
    o.total_price,
    o.status,
    u.name AS user_name,
    o.placed_at
FROM orders o
JOIN users u ON o.user_id = u.id
WHERE o.deleted_at IS NULL
ORDER BY o.placed_at DESC
LIMIT $1;

-- name: GetCategoryDistribution :many
SELECT 
    c.name as category_name,
    COUNT(p.id) as product_count
FROM categories c
LEFT JOIN products p ON c.id = p.category_id AND p.deleted_at IS NULL
WHERE c.deleted_at IS NULL
GROUP BY c.id, c.name
ORDER BY product_count DESC;
