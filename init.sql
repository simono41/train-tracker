CREATE TABLE IF NOT EXISTS trips (
    id VARCHAR(36) PRIMARY KEY,
    latitude DOUBLE,
    longitude DOUBLE,
    timestamp DATETIME,
    train_name VARCHAR(50),
    fahrt_nr VARCHAR(20),
    trip_id VARCHAR(255)
);
