INSERT INTO products (id, category_id, sku, name, brand, unit, price, weight_kg, requires_truck, is_active) VALUES
    ('prod_load_semen', 'cat_semen', 'LOAD-SMN-50KG', 'Semen Load Test 50kg', 'Tiga Roda', 'sak', 68000, 50, false, true),
    ('prod_load_low_stock', 'cat_semen', 'LOAD-SMN-LOW', 'Semen Low Stock Test 50kg', 'Tiga Roda', 'sak', 68000, 50, false, true)
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
    ('prod_load_semen', 1000000, 0),
    ('prod_load_low_stock', 5, 0)
ON CONFLICT (product_id) DO UPDATE
SET on_hand_qty = EXCLUDED.on_hand_qty,
    reserved_qty = EXCLUDED.reserved_qty,
    updated_at = now();
