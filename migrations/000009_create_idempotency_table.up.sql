CREATE TABLE idempotency_records (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    status VARCHAR(50) NOT NULL, 
    response_payload JSONB,  
    created_at TIMESTAMP DEFAULT NOW()
);