ALTER TABLE question
  ADD CONSTRAINT uk_question_exam_name UNIQUE (exam_id, name);
