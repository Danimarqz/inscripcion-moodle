ALTER TABLE exam
    ADD COLUMN percentile_group INT NULL,
    ADD INDEX idx_percentile_group (percentile_group);
