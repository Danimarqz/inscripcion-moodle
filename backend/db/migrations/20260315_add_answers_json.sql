-- Add answers_json column to user_exam_submission and backfill from user_answer

-- 1. Add column
ALTER TABLE user_exam_submission ADD COLUMN answers_json JSON DEFAULT NULL;

-- 2. Backfill from user_answer
UPDATE user_exam_submission ues
SET ues.answers_json = (
    SELECT JSON_OBJECTAGG(CAST(ua.question_id AS CHAR), ua.answer)
    FROM user_answer ua
    WHERE ua.submission_id = ues.id
)
WHERE ues.answers_json IS NULL;

-- 3. Verify (should return 0)
-- SELECT COUNT(*) FROM user_exam_submission WHERE answers_json IS NULL;
