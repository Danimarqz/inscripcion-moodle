-- 20260802_add_allow_merits_edit.sql
-- Permite que el alumno modifique sus méritos tantas veces como quiera. Por
-- defecto los méritos siguen siendo de una sola escritura.
-- AutoMigrate is disabled; apply manually.
ALTER TABLE exam
  ADD COLUMN allow_merits_edit BOOLEAN NOT NULL DEFAULT FALSE;
