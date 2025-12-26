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
  totalQuestions: null,
  submissionMessage: null,
  resultError: null,
  answersReview: null,
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
        totalQuestions: null,
        answersReview: null,
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
        totalQuestions: action.payload.totalQuestions,
        submissionMessage: action.payload.message,
        resultError: null,
        answersReview: action.payload.answersReview,
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
        totalQuestions: null,
        submissionMessage: null,
        resultError: action.payload ?? null,
        answersReview: null,
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
        totalQuestions: action.payload.totalQuestions,
        submissionMessage: action.payload.message,
        resultError: null,
        answersReview: action.payload.answersReview,
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
        totalQuestions: null,
        submissionMessage: null,
        resultError: action.payload ?? null,
        answersReview: null,
      };
    default:
      return state;
  }
};

export function useExamUi(initialState: ExamUiState = examUiInitialState) {
  return useReducer(examUiReducer, initialState);
}
