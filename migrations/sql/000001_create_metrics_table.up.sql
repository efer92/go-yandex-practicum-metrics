CREATE TABLE IF NOT EXISTS metrics (
    id    TEXT             NOT NULL,
    mtype TEXT             NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION,
    updated_at TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, mtype)
);
