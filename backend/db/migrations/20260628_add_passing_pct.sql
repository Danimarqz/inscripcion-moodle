-- Modo Xunta: per-group passing percentage (e.g. 40, 45). The grade at this
-- percentage is the group's min_passing_score; max_score is the grade at 100%.
ALTER TABLE question_group ADD COLUMN passing_pct DOUBLE NULL;
