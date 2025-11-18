-- =====================================================
-- DATABASE MIGRATION: order-service-sample
-- Context: Take Home Test - Backend Developer
-- Final Version (single warehouse per order)
-- =====================================================

-- USERS
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(100) UNIQUE NOT NULL,
    phone VARCHAR(20) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Seed user admin (password: admin123)
INSERT INTO users (email, phone, password_hash)
VALUES
  ('admin@example.com', '08123456789', '$2a$10$8bLxP2X2E0d0VnUq9ybT7emJvhIkHgKq06dKfXLRuHnvfIs0nU4Sa')
ON CONFLICT DO NOTHING;

-- PRODUCTS
CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price NUMERIC(12,2) NOT NULL,
    stock INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Seed products
INSERT INTO products (name, description, price, stock)
VALUES
  ('Wireless Mouse', 'High precision wireless mouse', 150000, 50),
  ('Mechanical Keyboard', 'RGB backlit mechanical keyboard', 700000, 30),
  ('USB-C Hub', '7-in-1 USB-C docking hub', 350000, 40),
  ('Gaming Headset', 'Noise cancelling over-ear headset', 450000, 25),
  ('Webcam HD', '1080p full HD USB webcam', 250000, 60)
ON CONFLICT DO NOTHING;

-- WAREHOUSES
CREATE TABLE IF NOT EXISTS warehouses (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Seed warehouses
INSERT INTO warehouses (name, active)
VALUES
  ('Central Warehouse', TRUE),
  ('Jakarta Distribution Center', TRUE),
  ('Surabaya Warehouse', TRUE)
ON CONFLICT DO NOTHING;

-- WAREHOUSE STOCK
CREATE TABLE IF NOT EXISTS warehouse_stock (
    id SERIAL PRIMARY KEY,
    warehouse_id INT NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    product_id INT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INT DEFAULT 0,
    reserved INT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (warehouse_id, product_id)
);

-- Seed warehouse stock
INSERT INTO warehouse_stock (warehouse_id, product_id, quantity, reserved)
VALUES
  (1, 1, 20, 0),
  (1, 2, 10, 0),
  (1, 3, 15, 0),
  (2, 4, 15, 0),
  (2, 5, 30, 0),
  (3, 1, 30, 0),
  (3, 2, 20, 0),
  (3, 3, 25, 0),
  (3, 4, 10, 0),
  (3, 5, 20, 0)
ON CONFLICT DO NOTHING;

-- ORDERS
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_amount NUMERIC(12,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ORDER ITEMS
CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id INT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INT NOT NULL REFERENCES products(id),
    quantity INT NOT NULL,
    price NUMERIC(12,2) NOT NULL
);

-- RESERVATIONS
CREATE TABLE IF NOT EXISTS reservations (
    id SERIAL PRIMARY KEY,
    order_id INT REFERENCES orders(id) ON DELETE CASCADE,
    product_id INT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    warehouse_id INT NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    quantity INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reservations_expires_at ON reservations (expires_at);

-- =====================================================
-- END OF MIGRATION
-- =====================================================
