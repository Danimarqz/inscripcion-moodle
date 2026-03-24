ALTER TABLE exam
  ADD COLUMN passing_threshold DOUBLE DEFAULT NULL;
-- Precomputed threshold value, calculated when passing criteria are set/updated.
