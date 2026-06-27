-- 20260627_add_wrong_block_size.sql
-- Adds block-penalty scoring: cada N falladas resta 1 pregunta entera de crédito,
-- en lugar de la penalización proporcional (0,25/fallo). NULL/0 = desactivado.
-- Excluyente con la penalización proporcional. En exámenes con grupos el conteo de
-- bloques es por grupo. AutoMigrate is disabled; apply manually.
ALTER TABLE exam
  ADD COLUMN wrong_block_size INT NULL;
