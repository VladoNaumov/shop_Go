-- Простая миграция: только таблица products
-- Удаляет зависимости и создаёт чистую таблицу

SET FOREIGN_KEY_CHECKS = 0;

-- Удаляет все связанные таблицы
DROP TABLE IF EXISTS product_discounts;
DROP TABLE IF EXISTS products;

SET FOREIGN_KEY_CHECKS = 1;

-- Простая таблица товаров
CREATE TABLE products (
 id VARCHAR(36) PRIMARY KEY,
 name VARCHAR(255) NOT NULL,
 article VARCHAR(100) UNIQUE NOT NULL,
 price DECIMAL(10,2) NOT NULL,
 image_alt VARCHAR(255),
 created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Индекс для поиска по артикулу
CREATE INDEX idx_products_article ON products(article);

-- Тестовые данные
INSERT INTO products (id, name, article, price, image_alt) VALUES
 ('1', 'Смартфон XYZ Pro', 'ART-001', 299.99, 'Смартфон с 128GB'),
 ('2', 'Ноутбук ABC Ultra', 'ART-002', 899.00, 'Ноутбук 16" i7'),
 ('3', 'Планшет DEF Mini', 'ART-003', 199.50, 'Планшет 10"'),
 ('4', 'Наушники GHI Wireless', 'ART-004', 79.90, 'Беспроводные TWS'),
 ('5', 'Клавиатура KLM Mechanical', 'ART-005', 129.00, NULL);