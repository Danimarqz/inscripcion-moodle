import { ANSWER_OPTIONS as COMMON_ANSWER_OPTIONS } from '../../constants/answerOptions';
import type { AnswerOption as COMMON_ANSWER_OPTION } from '../../constants/answerOptions';
import type { SubmissionBreakdown } from '../../types/exam';

export const ANSWER_OPTIONS = COMMON_ANSWER_OPTIONS;
export type AnswerOption = COMMON_ANSWER_OPTION;

export type SubmissionOrderBy = 'submitted_at' | 'score' | 'name' | 'surname';
export type SubmissionOrderDir = 'asc' | 'desc';

export interface EditingState {
  submissionId: number;
  name: string;
  surname: string;
  email: string;
  dni: string;
  answers: Record<number, AnswerOption | '-'>;
  merits?: number;
  breakdown?: SubmissionBreakdown | null;
}
