-- 1. Insert a core Shipment
INSERT INTO shipments (
    shipment_id, carrier, tracking_number, origin_location_id,
    destination_location_id, status, eta, dispatched_at, created_at, updated_at
) VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'InternalFleet',
    'TRK-99283411',
    'WH-JHB-001',
    'DC-CPT-002',
    'IN_TRANSIT',
    NOW() + INTERVAL '2 days',
    NOW() - INTERVAL '4 hours',
    NOW() - INTERVAL '1 day',
    NOW()
);

-- 2. Insert Shipment Items
INSERT INTO shipment_items (shipment_id, sku, quantity) VALUES
('550e8400-e29b-41d4-a716-446655440000', 'SKU-LAPTOP-PRO-14', 10),
('550e8400-e29b-41d4-a716-446655440000', 'SKU-MONITOR-4K-27', 5),
('550e8400-e29b-41d4-a716-446655440000', 'SKU-CABLE-USB-C', 50);

-- 3. Insert Status History (Audit Trail)
-- Note: 'seq' is BIGSERIAL, so we let the DB handle it or specify if needed.
INSERT INTO shipment_status_history (shipment_id, old_status, new_status, reason, changed_at) VALUES
('550e8400-e29b-41d4-a716-446655440000', 'CREATED', 'PICKED_UP', 'Package loaded at Warehouse', NOW() - INTERVAL '5 hours'),
('550e8400-e29b-41d4-a716-446655440000', 'PICKED_UP', 'IN_TRANSIT', 'Driver departed warehouse', NOW() - INTERVAL '4 hours');

-- 4. Insert GPS Tracking Logs
-- Simulated movement along a route
INSERT INTO shipment_tracking (shipment_id, latitude, longitude, recorded_at) VALUES
('550e8400-e29b-41d4-a716-446655440000', -26.2041, 28.0473, NOW() - INTERVAL '4 hours'),   -- Johannesburg
('550e8400-e29b-41d4-a716-446655440000', -29.1181, 26.2235, NOW() - INTERVAL '2 hours'),   -- Bloemfontein
('550e8400-e29b-41d4-a716-446655440000', -30.5960, 25.5030, NOW() - INTERVAL '30 minutes'); -- Colesberg
