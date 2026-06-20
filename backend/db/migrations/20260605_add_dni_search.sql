-- 20260605_add_dni_search.sql
-- Adds an indexable search column to exam_official_result so the official-result
-- lookup stops doing a full table scan (~9000 rows/exam loaded into Go) per request.
--
-- dni_masked stores the masked DNI with asterisk fill chars (e.g. "****2611*") and
-- inconsistent patterns, so LIKE '%...%' has a leading wildcard and cannot use an
-- index. dni_search is a STORED generated column that strips every non-digit, leaving
-- only the real digits, which IS indexable for an IN(...)/seek lookup.
--
-- AutoMigrate is disabled; apply manually. The STORED column backfills existing rows
-- automatically on ADD COLUMN (verified: ~36000 rows populated).
ALTER TABLE exam_official_result
  ADD COLUMN dni_search VARCHAR(16)
    GENERATED ALWAYS AS (REGEXP_REPLACE(dni_masked, '[^0-9]', '')) STORED;

-- Composite B-tree index turns the per-exam scan into a seek.
CREATE INDEX idx_official_result_exam_dnisearch
  ON exam_official_result (exam_id, dni_search);
