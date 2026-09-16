CREATE TABLE IF NOT EXISTS clients (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    client_id       VARCHAR(255),
    client_secret   TEXT,
    usertoken       TEXT,
    passtoken       TEXT,
    auth_token      TEXT,
    refresh_token   TEXT,
    base_url        VARCHAR(512),
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS master_violations (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(50) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    raw_data    JSONB,
    synced_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(code)
);

CREATE TABLE IF NOT EXISTS cameras (
    id              SERIAL PRIMARY KEY,
    client_id       INT REFERENCES clients(id) ON DELETE CASCADE,
    camera_code     VARCHAR(100),
    device_name     VARCHAR(255) NOT NULL,
    location_name   TEXT,
    address         TEXT,
    latitude        DOUBLE PRECISION,
    longitude       DOUBLE PRECISION,
    status          VARCHAR(50) DEFAULT 'active',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(client_id, device_name)
);

ALTER TABLE cameras ADD COLUMN IF NOT EXISTS camera_code VARCHAR(100);
ALTER TABLE cameras ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE cameras ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_cameras_camera_code ON cameras(camera_code);
CREATE INDEX IF NOT EXISTS idx_cameras_deleted ON cameras(deleted_at);

ALTER TABLE clients ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_clients_deleted ON clients(deleted_at);

CREATE TABLE IF NOT EXISTS violations (
    id                SERIAL PRIMARY KEY,
    client_id         INT REFERENCES clients(id),
    camera_id         INT REFERENCES cameras(id),
    device_name       VARCHAR(255),
    plate             VARCHAR(50),
    plate_color       VARCHAR(50),
    plate_image_url   TEXT,
    vehicle_type      VARCHAR(50),
    vehicle_color     VARCHAR(50),
    vehicle_image_url TEXT,
    video_url         TEXT,
    violation_code    VARCHAR(50),
    violation_name    VARCHAR(255),
    location_name     TEXT,
    capture_time      BIGINT,
    status            VARCHAR(50) DEFAULT 'pending',
    response_status   INT,
    attempts          INT DEFAULT 0,
    error_message     TEXT,
    sent_at           TIMESTAMPTZ,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    username      VARCHAR(100) UNIQUE NOT NULL,
    email         VARCHAR(255) UNIQUE,
    password_hash TEXT NOT NULL,
    name          VARCHAR(255),
    role          VARCHAR(50) DEFAULT 'operator',
    is_active     BOOLEAN DEFAULT TRUE,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sync_logs (
    id              SERIAL PRIMARY KEY,
    client_id       INT REFERENCES clients(id) ON DELETE SET NULL,
    action          VARCHAR(100) DEFAULT 'sync_master',
    status          VARCHAR(50) NOT NULL,
    records_synced  INT DEFAULT 0,
    message         TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_violations_status ON violations(status);
CREATE INDEX IF NOT EXISTS idx_violations_client ON violations(client_id);
CREATE INDEX IF NOT EXISTS idx_violations_created ON violations(created_at);
CREATE INDEX IF NOT EXISTS idx_violations_pending_realtime ON violations(created_at, attempts) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_sync_logs_created ON sync_logs(created_at DESC);
