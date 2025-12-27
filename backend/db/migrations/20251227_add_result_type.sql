-- Add result_type to exam_official_result and user_exam_submission

ALTER TABLE exam_official_result ADD COLUMN result_type VARCHAR(50) DEFAULT 'General' NOT NULL;
ALTER TABLE user_exam_submission ADD COLUMN selected_result_type VARCHAR(50) DEFAULT 'General' NOT NULL;
