import { useReducer, type Reducer } from 'preact/hooks';

import type { ExamUiAction, ExamUiState } from '../types/exam';

export const examUiInitialState: ExamUiState = {
  checking: false,
  hasPreviousSubmission: false,
  score: null,
  percentile: null,
  position: null,
  totalSubmissions: null,
  correctAnswers: null,
  incorrectAnswers: null,
  notAnswered: null,
  totalQuestions: null,
  submissionMessage: null,
  resultError: null,
  answersReview: null,
  maxScore: null,
  secondaryMaxScores: null,
  isPassed: null,
  canEditMerits: false,
  merits: null,
  meritsPosition: null,
  meritsTotal: null,
  passedCount: null,
};

export const examUiReducer: Reducer<ExamUiState, ExamUiAction> = (state, action) => {
  switch (action.type) {
    case 'RESET':
      return { ...examUiInitialState };
    case 'CHECK_START':
      return {
        ...state,
        checking: true,
        submissionMessage: null,
        resultError: null,
        score: null,
        percentile: null,
        position: null,
        totalSubmissions: null,
        correctAnswers: null,
        incorrectAnswers: null,
        notAnswered: null,
        totalQuestions: null,
        answersReview: null,
        maxScore: null,
        secondaryMaxScores: null,
        isPassed: null,
        canEditMerits: false,
        merits: null,
        meritsPosition: null,
        meritsTotal: null,
        passedCount: null,
      };
    case 'CHECK_SUCCESS':
      return {
        checking: false,
        hasPreviousSubmission: true,
        score: action.payload.score,
        percentile: action.payload.percentile,
        position: action.payload.position,
        totalSubmissions: action.payload.totalSubmissions,
        correctAnswers: action.payload.correctAnswers,
        incorrectAnswers: action.payload.incorrectAnswers,
        notAnswered: action.payload.notAnswered,
        totalQuestions: action.payload.totalQuestions,
        submissionMessage: action.payload.message,
        resultError: null,
        answersReview: action.payload.answersReview,
        maxScore: action.payload.max_score ?? null,
        secondaryMaxScores: action.payload.secondary_max_scores ?? null,
        isPassed: action.payload.isPassed,
        canEditMerits: action.payload.canEditMerits,
        merits: action.payload.merits,
        meritsPosition: action.payload.meritsPosition,
        meritsTotal: action.payload.meritsTotal,
        passedCount: action.payload.passedCount,
      };
    case 'CHECK_ERROR':
      return {
        checking: false,
        hasPreviousSubmission: false,
        score: null,
        percentile: null,
        position: null,
        totalSubmissions: null,
        correctAnswers: null,
        incorrectAnswers: null,
        notAnswered: null,
        totalQuestions: null,
        submissionMessage: null,
        resultError: action.payload ?? null,
        answersReview: null,
        maxScore: null,
        secondaryMaxScores: null,
        isPassed: null,
        canEditMerits: false,
        merits: null,
        meritsPosition: null,
        meritsTotal: null,
        passedCount: null,
      };
    case 'SUBMIT_SUCCESS':
      return {
        checking: false,
        hasPreviousSubmission: true,
        score: action.payload.score,
        percentile: action.payload.percentile,
        position: action.payload.position,
        totalSubmissions: action.payload.totalSubmissions,
        correctAnswers: action.payload.correctAnswers,
        incorrectAnswers: action.payload.incorrectAnswers,
        notAnswered: action.payload.notAnswered,
        totalQuestions: action.payload.totalQuestions,
        submissionMessage: action.payload.message,
        resultError: null,
        answersReview: action.payload.answersReview,
        maxScore: action.payload.max_score ?? null,
        secondaryMaxScores: action.payload.secondary_max_scores ?? null,
        isPassed: action.payload.isPassed,
        canEditMerits: action.payload.canEditMerits,
        merits: action.payload.merits,
        meritsPosition: action.payload.meritsPosition,
        meritsTotal: action.payload.meritsTotal,
        passedCount: action.payload.passedCount,
      };
    case 'SUBMIT_ERROR':
      return {
        ...state,
        checking: false,
        hasPreviousSubmission: false,
        score: null,
        percentile: null,
        position: null,
        totalSubmissions: null,
        correctAnswers: null,
        incorrectAnswers: null,
        notAnswered: null,
        totalQuestions: null,
        submissionMessage: null,
        resultError: action.payload ?? null,
        answersReview: null,
        maxScore: null,
        secondaryMaxScores: null,
        isPassed: null,
        canEditMerits: false,
        merits: null,
        meritsPosition: null,
        meritsTotal: null,
        passedCount: null,
      };
    default:
      return state;
  }
};

export function useExamUi(initialState: ExamUiState = examUiInitialState) {
  return useReducer(examUiReducer, initialState);
}
