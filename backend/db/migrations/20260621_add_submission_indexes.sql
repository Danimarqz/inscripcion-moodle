-- user_exam_submission only had single-column indexes on exam_id and user_id.
-- The admin list filters by exam_id and orders by submitted_at (filesort today);
-- stats/position/passed-count filter by exam_id + score (scan today).
-- These composites cover both: index range-scan instead of filesort/full scan.
CREATE INDEX idx_submission_exam_submitted ON user_exam_submission (exam_id, submitted_at);
CREATE INDEX idx_submission_exam_score ON user_exam_submission (exam_id, score);
