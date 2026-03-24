import type {
  AnswerReview,
  ExamOut,
  ExamResultPayload,
} from '../types/exam';
import { roundToTwoDecimals } from './validation';

type ComposeMessageParams = {
  baseMessage?: string | null;
  showScore: boolean;
  showPercentile: boolean;
  showScoreFull: boolean;
  score: number | null;
  percentile: number | null;
  position: number | null;
  totalSubmissions: number | null;
  correctAnswers: number | null;
  totalQuestions: number | null;
};

export function composeResultMessage({
  baseMessage,
  showScore,
  showPercentile,
  showScoreFull,
  score,
  percentile,
  position,
  totalSubmissions,
  correctAnswers,
  totalQuestions,
}: ComposeMessageParams): string {
  const parts: string[] = [];
  if (baseMessage) {
    parts.push(baseMessage.trim());
  }
  if (showScore && score !== null) {
    parts.push(`Tu puntuacion es ${score}`);
  }
  if (showScoreFull && correctAnswers !== null && totalQuestions !== null) {
    parts.push(`Has acertado ${correctAnswers} de ${totalQuestions} preguntas`);
  }
  if (showPercentile && percentile !== null) {
    let percentileMessage = `Tienes el percentil ${percentile}`;
    if (position !== null && totalSubmissions !== null) {
      percentileMessage += `, estas en la posicion ${position} de ${totalSubmissions}`;
    }
    parts.push(percentileMessage);
  }

  const result = parts.filter(Boolean).join('. ').trim();
  return result || 'Tu entrega ha sido registrada correctamente';
}

export function buildResultPayload(
  result: ExamOut,
  context: 'check' | 'submit',
  config: { showScore: boolean; showPercentile: boolean; showScoreFull: boolean }
): ExamResultPayload {
  const { showScore, showPercentile, showScoreFull } = config;

  const baseMessage =
    result.message ??
    (context === 'check'
      ? 'Ya has entregado este examen anteriormente'
      : 'Tu entrega ha sido registrada correctamente');

  const nextScore =
    showScore && typeof result.score === 'number' ? roundToTwoDecimals(result.score) : null;
  const nextPercentile =
    showPercentile && typeof result.percentile === 'number'
      ? roundToTwoDecimals(result.percentile)
      : null;
  const rawPosition = typeof result.position === 'number' ? result.position : null;
  const rawTotalSubmissions =
    typeof result.total_submissions === 'number' ? result.total_submissions : null;
  const rawCorrectAnswers =
    typeof result.correct_answers === 'number' ? result.correct_answers : null;
  const rawIncorrectAnswers =
    typeof result.incorrect_answers === 'number' ? result.incorrect_answers : null;
  const rawNotAnswered =
    typeof result.not_answered === 'number' ? result.not_answered : null;
  const rawTotalQuestions =
    typeof result.total_questions === 'number' ? result.total_questions : null;
  const review =
    Array.isArray(result.answers_review) && result.answers_review.length > 0
      ? (result.answers_review as AnswerReview[])
      : null;

  const nextPosition = showPercentile ? rawPosition : null;
  const nextTotalSubmissions = showPercentile ? rawTotalSubmissions : null;
  const nextCorrectAnswers = showScoreFull ? rawCorrectAnswers : null;
  const nextIncorrectAnswers = showScoreFull ? rawIncorrectAnswers : null;
  const nextNotAnswered = showScoreFull ? rawNotAnswered : null;
  const nextTotalQuestions = showScoreFull ? rawTotalQuestions : null;

  const message = composeResultMessage({
    baseMessage,
    showScore,
    showPercentile,
    showScoreFull,
    score: nextScore,
    percentile: nextPercentile,
    position: nextPosition,
    totalSubmissions: nextTotalSubmissions,
    correctAnswers: nextCorrectAnswers,
    totalQuestions: nextTotalQuestions,
  });

  return {
    score: nextScore,
    percentile: nextPercentile,
    position: nextPosition,
    totalSubmissions: nextTotalSubmissions,
    correctAnswers: nextCorrectAnswers,
    incorrectAnswers: nextIncorrectAnswers,
    notAnswered: nextNotAnswered,
    totalQuestions: nextTotalQuestions,
    message,
    answersReview: review,
    max_score: config.showScore ? result.max_score : undefined,
    secondary_max_scores: config.showScore ? result.secondary_max_scores : undefined,
    isPassed: result.is_passed ?? null,
    canEditMerits: result.can_edit_merits ?? false,
    merits: result.merits ?? null,
    meritsPosition: result.merits_position ?? null,
    meritsTotal: result.merits_total ?? null,
    passedCount: result.passed_count ?? null,
  };
}
