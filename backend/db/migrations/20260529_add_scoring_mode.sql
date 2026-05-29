-- 20260529_add_scoring_mode.sql
-- Adds ABSOLUTE scoring mode support. Existing exams remain LEGACY.
-- AutoMigrate is disabled; apply manually.
-- MUST be applied BEFORE deploying code that issues GORM Save on the exam table,
-- so no pre-migration row can be re-saved with an empty scoring_mode.
ALTER TABLE exam
    ADD COLUMN scoring_mode       VARCHAR(20) NOT NULL DEFAULT 'legacy',
    ADD COLUMN points_per_correct DOUBLE      NULL,
    ADD COLUMN points_per_wrong   DOUBLE      NULL;

-- Idempotent backfill for any row that somehow got an empty value.
UPDATE exam SET scoring_mode = 'legacy' WHERE scoring_mode = '';
