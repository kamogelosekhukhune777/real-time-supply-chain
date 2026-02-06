CREATE TABLE shipments (
  shipment_id     TEXT PRIMARY KEY,                 -- uuid
  carrier         TEXT NOT NULL,                     -- "DHL", "FedEx", "InternalFleet"
  tracking_number TEXT UNIQUE,                       -- optional depending on carrier
  origin_location_id      TEXT NOT NULL,             -- warehouse / supplier location
  destination_location_id TEXT NOT NULL,             -- warehouse / customer zone
  status          TEXT NOT NULL,                     -- CREATED, PICKED_UP, IN_TRANSIT, DELAYED, DELIVERED, CANCELLED
  eta             TIMESTAMP,                         -- predicted arrival time
  dispatched_at   TIMESTAMP,
  delivered_at    TIMESTAMP,
  created_at      TIMESTAMP NOT NULL DEFAULT now(),
  updated_at      TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE shipment_items (
  shipment_id TEXT NOT NULL,
  sku         TEXT NOT NULL,
  quantity    INTEGER NOT NULL,

  PRIMARY KEY (shipment_id, sku),
  CONSTRAINT fk_shipment_items_shipment
    FOREIGN KEY (shipment_id) REFERENCES shipments (shipment_id)
    ON DELETE CASCADE,

  CHECK (quantity > 0)
);

CREATE TABLE shipment_tracking (
  shipment_id TEXT NOT NULL,
  seq         BIGSERIAL,                             -- monotonic event ordering
  latitude    DOUBLE PRECISION NOT NULL,
  longitude   DOUBLE PRECISION NOT NULL,
  recorded_at TIMESTAMP NOT NULL DEFAULT now(),

  PRIMARY KEY (shipment_id, seq),

  CONSTRAINT fk_shipment_tracking_shipment
    FOREIGN KEY (shipment_id) REFERENCES shipments (shipment_id)
    ON DELETE CASCADE,

  CHECK (latitude >= -90 AND latitude <= 90),
  CHECK (longitude >= -180 AND longitude <= 180)
);

CREATE TABLE shipment_status_history (
  shipment_id TEXT NOT NULL,
  seq         BIGSERIAL,
  old_status  TEXT NOT NULL,
  new_status  TEXT NOT NULL,
  reason      TEXT,                                  -- "WEATHER_DELAY", "CUSTOMS", "TRUCK_BREAKDOWN"
  changed_at  TIMESTAMP NOT NULL DEFAULT now(),

  PRIMARY KEY (shipment_id, seq),

  CONSTRAINT fk_shipment_status_history_shipment
    FOREIGN KEY (shipment_id) REFERENCES shipments (shipment_id)
    ON DELETE CASCADE
);

CREATE INDEX idx_shipments_status ON shipments(status);
CREATE INDEX idx_shipments_origin ON shipments(origin_location_id);
CREATE INDEX idx_shipments_destination ON shipments(destination_location_id);
CREATE INDEX idx_shipments_eta ON shipments(eta);

CREATE INDEX idx_tracking_recent
ON shipment_tracking(shipment_id, recorded_at DESC);
