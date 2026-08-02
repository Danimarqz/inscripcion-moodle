import { describe, it, expect } from 'vitest';
import { composeResultMessage, buildResultPayload } from '../examLogic';

describe('composeResultMessage', () => {
  it('returns default', () => {
    const msg = composeResultMessage({
      baseMessage: null, showScore: false, showPercentile: false, showScoreFull: false,
      score: null, percentile: null, position: null, totalSubmissions: null,
      correctAnswers: null, totalQuestions: null,
    });
    expect(msg).toBe('Tu entrega ha sido registrada correctamente');
  });

  it('shows score', () => {
    const msg = composeResultMessage({
      baseMessage: null, showScore: true, showPercentile: false, showScoreFull: false,
      score: 7.5, percentile: null, position: null, totalSubmissions: null,
      correctAnswers: null, totalQuestions: null,
    });
    expect(msg).toContain('Tu puntuacion es 7.5');
  });

  it('hides score', () => {
    const msg = composeResultMessage({
      baseMessage: null, showScore: false, showPercentile: false, showScoreFull: false,
      score: 7.5, percentile: null, position: null, totalSubmissions: null,
      correctAnswers: null, totalQuestions: null,
    });
    expect(msg).toBe('Tu entrega ha sido registrada correctamente');
  });

  it('shows percentile', () => {
    const msg = composeResultMessage({
      baseMessage: null, showScore: false, showPercentile: true, showScoreFull: false,
      score: null, percentile: 85, position: 10, totalSubmissions: 200,
      correctAnswers: null, totalQuestions: null,
    });
    expect(msg).toContain('percentil 85');
    expect(msg).toContain('posicion 10');
  });

  it('shows correct answers', () => {
    const msg = composeResultMessage({
      baseMessage: null, showScore: false, showPercentile: false, showScoreFull: true,
      score: null, percentile: null, position: null, totalSubmissions: null,
      correctAnswers: 15, totalQuestions: 20,
    });
    expect(msg).toContain('acertado 15 de 20');
  });

  it('combines parts', () => {
    const msg = composeResultMessage({
      baseMessage: 'Aviso', showScore: true, showPercentile: false, showScoreFull: true,
      score: 6.0, percentile: null, position: null, totalSubmissions: null,
      correctAnswers: 12, totalQuestions: 20,
    });
    expect(msg).toBe('Aviso. Tu puntuacion es 6. Has acertado 12 de 20 preguntas');
  });

  it('null score hidden', () => {
    const msg = composeResultMessage({
      baseMessage: null, showScore: true, showPercentile: false, showScoreFull: false,
      score: null, percentile: null, position: null, totalSubmissions: null,
      correctAnswers: null, totalQuestions: null,
    });
    expect(msg).toBe('Tu entrega ha sido registrada correctamente');
  });

  it('percentile without position', () => {
    const msg = composeResultMessage({
      baseMessage: null, showScore: false, showPercentile: true, showScoreFull: false,
      score: null, percentile: 90, position: null, totalSubmissions: null,
      correctAnswers: null, totalQuestions: null,
    });
    expect(msg).toContain('percentil 90');
    expect(msg).not.toContain('posicion');
  });
});

describe('buildResultPayload', () => {
  const out = {
    score: 7.5, percentile: 85, position: 10, total_submissions: 200,
    correct_answers: 15, incorrect_answers: 3, not_answered: 2, total_questions: 20,
    is_passed: true, max_score: 10, groups: [],
  };

  it('maps all', () => {
    const r = buildResultPayload(out as any, 'submit', {
      showScore: true, showPercentile: true, showScoreFull: true,
    });
    expect(r.score).toBe(7.5);
    expect(r.percentile).toBe(85);
    expect(r.position).toBe(10);
    expect(r.totalSubmissions).toBe(200);
    expect(r.correctAnswers).toBe(15);
    expect(r.incorrectAnswers).toBe(3);
    expect(r.notAnswered).toBe(2);
    expect(r.totalQuestions).toBe(20);
    expect(r.isPassed).toBe(true);
    expect(r.max_score).toBe(10);
    expect(r.message).toContain('puntuacion');
  });

  it('nulls score when hidden', () => {
    const r = buildResultPayload(out as any, 'submit', {
      showScore: false, showPercentile: true, showScoreFull: true,
    });
    expect(r.score).toBeNull();
    expect(r.max_score).toBeUndefined();
  });

  it('nulls percentiles when hidden', () => {
    const r = buildResultPayload(out as any, 'submit', {
      showScore: true, showPercentile: false, showScoreFull: true,
    });
    expect(r.percentile).toBeNull();
    expect(r.position).toBeNull();
    expect(r.totalSubmissions).toBeNull();
  });

  it('nulls answers when hidden', () => {
    const r = buildResultPayload(out as any, 'submit', {
      showScore: true, showPercentile: true, showScoreFull: false,
    });
    expect(r.correctAnswers).toBeNull();
    expect(r.incorrectAnswers).toBeNull();
    expect(r.notAnswered).toBeNull();
    expect(r.totalQuestions).toBeNull();
  });

  it('handles official only', () => {
    const off = { ...out, is_official_only: true, correct_answers: undefined, incorrect_answers: undefined, not_answered: undefined, total_questions: undefined, answers_review: [] };
    const r = buildResultPayload(off as any, 'submit', {
      showScore: true, showPercentile: true, showScoreFull: true,
    });
    expect(r.score).toBe(7.5);
    expect(r.correctAnswers).toBeNull();
    expect(r.answersReview).toBeNull();
  });

  it('has check default', () => {
    const r = buildResultPayload({} as any, 'check', {
      showScore: false, showPercentile: false, showScoreFull: false,
    });
    expect(r.message).toBe('Ya has entregado este examen anteriormente');
  });

  it('has submit default', () => {
    const r = buildResultPayload({} as any, 'submit', {
      showScore: false, showPercentile: false, showScoreFull: false,
    });
    expect(r.message).toBe('Tu entrega ha sido registrada correctamente');
  });

  it('groups hidden when score hidden', () => {
    const r = buildResultPayload({ groups: [{ id: 1 }] } as any, 'submit', {
      showScore: false, showPercentile: false, showScoreFull: false,
    });
    expect(r.groups).toBeNull();
  });

  it('passes groups when score shown', () => {
    const groups = [{ id: 1, name: 'Teorico' }];
    const r = buildResultPayload({ ...out, groups } as any, 'submit', {
      showScore: true, showPercentile: false, showScoreFull: false,
    });
    expect(r.groups).toEqual(groups);
  });
});
