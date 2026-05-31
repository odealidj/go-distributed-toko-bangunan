INSERT INTO categories (id, name, is_active) VALUES
    ('cat_semen', 'Semen', true),
    ('cat_besi', 'Besi', true),
    ('cat_cat', 'Cat', true),
    ('cat_pasir', 'Pasir', true)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    is_active = EXCLUDED.is_active;

INSERT INTO products (id, category_id, sku, name, brand, unit, price, weight_kg, requires_truck, is_active) VALUES
    ('prod_semen_50kg', 'cat_semen', 'SMN-50KG', 'Semen Portland 50kg', 'Tiga Roda', 'sak', 68000, 50, false, true),
    ('prod_besi_10mm', 'cat_besi', 'BSI-10MM', 'Besi Beton 10mm', 'KS', 'batang', 72000, 7.4, false, true),
    ('prod_cat_putih_5kg', 'cat_cat', 'CAT-PTH-5KG', 'Cat Tembok Putih 5kg', 'Dulux', 'pail', 145000, 5, false, true),
    ('prod_pasir_1m3', 'cat_pasir', 'PSR-1M3', 'Pasir Beton 1m3', NULL, 'm3', 310000, 1500, true, true)
ON CONFLICT (id) DO UPDATE
SET category_id = EXCLUDED.category_id,
    sku = EXCLUDED.sku,
    name = EXCLUDED.name,
    brand = EXCLUDED.brand,
    unit = EXCLUDED.unit,
    price = EXCLUDED.price,
    weight_kg = EXCLUDED.weight_kg,
    requires_truck = EXCLUDED.requires_truck,
    is_active = EXCLUDED.is_active;

INSERT INTO inventories (product_id, on_hand_qty, reserved_qty) VALUES
    ('prod_semen_50kg', 120, 0),
    ('prod_besi_10mm', 80, 0),
    ('prod_cat_putih_5kg', 45, 0),
    ('prod_pasir_1m3', 12, 0)
ON CONFLICT (product_id) DO UPDATE
SET on_hand_qty = EXCLUDED.on_hand_qty,
    reserved_qty = EXCLUDED.reserved_qty,
    updated_at = now();

