export const ANSWER_OPTIONS = ['A', 'B', 'C', 'D'] as const;
export type AnswerOption = (typeof ANSWER_OPTIONS)[number];
