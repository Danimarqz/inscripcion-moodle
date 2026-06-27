-- 20260626_add_raffle_terms.sql
-- Adds per-exam raffle terms: a toggle and a long-text field. When enabled, the
-- public exam page forces the student to accept the terms before submitting.
-- AutoMigrate is disabled; apply manually.
ALTER TABLE exam
  ADD COLUMN raffle_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN raffle_terms TEXT NOT NULL DEFAULT '';
