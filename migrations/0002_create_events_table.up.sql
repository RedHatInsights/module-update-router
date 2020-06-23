CREATE TABLE events (
    event_id VARCHAR(36) PRIMARY KEY,
    phase VARCHAR(256) NOT NULL,
    started_at TIMESTAMP NOT NULL,
    exit INTEGER NOT NULL,
    exception VARCHAR(1024),
    duration INTEGER NOT NULL,
    machine_id VARCHAR(36) NOT NULL,
    core_version VARCHAR(256) NOT NULL
);
