ALTER TABLE exam
  ADD COLUMN passing_criteria_type VARCHAR(20) NOT NULL DEFAULT 'disabled',
  ADD COLUMN passing_criteria_value DOUBLE DEFAULT NULL;
-- passing_criteria_type: 'disabled' | 'min_score' | 'top10_pct'
