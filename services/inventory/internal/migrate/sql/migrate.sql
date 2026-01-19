CREATE TABLE inventory (
  location_id TEXT NOT NULL,
  sku         TEXT NOT NULL,

  on_hand     INTEGER NOT NULL DEFAULT 0,
  reserved    INTEGER NOT NULL DEFAULT 0,
  incoming    INTEGER NOT NULL DEFAULT 0,

  updated_at  TIMESTAMP NOT NULL DEFAULT now(),

  PRIMARY KEY (location_id, sku),
  CHECK (on_hand >= 0),
  CHECK (reserved >= 0),
  CHECK (incoming >= 0),
  CHECK (reserved <= on_hand)
);
