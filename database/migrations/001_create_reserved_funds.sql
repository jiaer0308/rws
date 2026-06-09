-- Migration: Create file_batches and group_reserved_funds tables

CREATE TABLE IF NOT EXISTS file_batches (
    id SERIAL PRIMARY KEY,
    batch_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    original_file_path TEXT NULL,
    result_file_path TEXT NULL,
    error_report_path TEXT NULL,
    total_rows INT NOT NULL DEFAULT 0,
    success_rows INT NOT NULL DEFAULT 0,
    error_rows INT NOT NULL DEFAULT 0,
    created_by_user_id INT NULL,
    created_by_name VARCHAR(255) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS group_reserved_funds (
    id SERIAL PRIMARY KEY,
    refund_date DATE NOT NULL,
    fb_name VARCHAR(255) NOT NULL,
    bank VARCHAR(255) NOT NULL,
    reserved_funds_date DATE NOT NULL,
    bank_order_detail VARCHAR(255) NOT NULL,
    ar_payment_no VARCHAR(255) NOT NULL UNIQUE,
    deducted_order_no VARCHAR(255) NULL,
    amt NUMERIC(10,2) NOT NULL,
    asst_name VARCHAR(255) NOT NULL,
    source_import_batch_id INT NULL REFERENCES file_batches(id),
    matched_export_batch_id INT NULL REFERENCES file_batches(id),
    matched_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexing for optimized matching
CREATE INDEX IF NOT EXISTS idx_group_reserved_funds_lookup ON group_reserved_funds (
    LOWER(TRIM(fb_name)), 
    LOWER(TRIM(bank_order_detail)), 
    reserved_funds_date, 
    amt
);
