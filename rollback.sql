-- ==========================================
-- ROLLBACK SCRIPT
-- Drop tables in correct dependency order
-- ==========================================

DROP TABLE IF EXISTS order_items CASCADE;

DROP TABLE IF EXISTS reservations CASCADE;

DROP TABLE IF EXISTS warehouse_stock CASCADE;

DROP TABLE IF EXISTS orders CASCADE;

DROP TABLE IF EXISTS products CASCADE;

DROP TABLE IF EXISTS warehouses CASCADE;

DROP TABLE IF EXISTS users CASCADE;
