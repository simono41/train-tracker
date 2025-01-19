-- Tabelle für Zugfahrten
CREATE TABLE IF NOT EXISTS trips (
    id VARCHAR(36) PRIMARY KEY,
    latitude DOUBLE,
    longitude DOUBLE,
    timestamp DATETIME,
    train_name VARCHAR(50),
    fahrt_nr VARCHAR(20),
    trip_id VARCHAR(255),
    planned_timestamp DATETIME,
    delay INT,
    INDEX idx_fahrt_nr_timestamp (fahrt_nr, timestamp)
);

-- Tabelle für tägliche Verspätungsstatistiken
CREATE TABLE IF NOT EXISTS today_delay_stats (
    id VARCHAR(36) PRIMARY KEY,
    fahrt_nr VARCHAR(255),
    train_name VARCHAR(255),
    delay INT,
    timestamp DATETIME,
    UNIQUE KEY uk_fahrt_nr_date (fahrt_nr, timestamp)
);

-- Tabelle für aggregierte Verspätungsstatistiken
CREATE TABLE IF NOT EXISTS delay_stats (
    id VARCHAR(36) PRIMARY KEY,
    fahrt_nr VARCHAR(255),
    total_trips INT,
    delayed_trips INT,
    avg_delay FLOAT,
    last_updated DATETIME,
    INDEX idx_fahrt_nr (fahrt_nr)
);
