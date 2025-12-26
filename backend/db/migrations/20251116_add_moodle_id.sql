-- Add a nullable moodle_id reference to keep track of Moodle users associated with exam_user.
ALTER TABLE exam_user
ADD COLUMN moodle_id INT NULL AFTER email;
