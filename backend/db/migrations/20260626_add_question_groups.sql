-- 20260626_add_question_groups.sql
-- Grupos de preguntas con puntuacion por grupo. Grupos opcionales: un examen sin
-- grupos mantiene el comportamiento plano actual. AutoMigrate deshabilitado:
-- aplicar a mano. Sin FOREIGN KEY (convencion del proyecto).
CREATE TABLE question_group (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    exam_id           BIGINT UNSIGNED NOT NULL,
    name              VARCHAR(64)     NOT NULL,
    position          INT             NOT NULL DEFAULT 0,
    max_score         DOUBLE          NOT NULL,
    points_per_wrong  DOUBLE          NOT NULL DEFAULT 0,
    min_passing_score DOUBLE          NULL,
    eliminatory       TINYINT(1)      NOT NULL DEFAULT 0,
    INDEX idx_question_group_exam (exam_id)
);

ALTER TABLE question
    ADD COLUMN group_id BIGINT UNSIGNED NULL AFTER exam_id,
    ADD INDEX idx_question_group_id (group_id);
