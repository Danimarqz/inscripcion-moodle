-- Official scores now apply automatically whenever a linked official result
-- has a score (COALESCE in score/percentile/ranking), so the per-exam toggle
-- is no longer used.
ALTER TABLE exam DROP COLUMN use_official_scores;
