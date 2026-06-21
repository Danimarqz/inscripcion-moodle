-- exam_official_result.user_id had no index. Every official-score JOIN keys on
-- (exam_id, user_id): percentile recalc, average, merits ranking, score override.
-- This composite turns those nested-loop scans into index lookups.
CREATE INDEX idx_official_result_exam_user ON exam_official_result (exam_id, user_id);
