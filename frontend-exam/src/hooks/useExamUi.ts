import { useReducer, type Reducer } from 'preact/hooks';

import type { ExamUiAction, ExamUiState } from '../types/exam';

export const examUiInitialState: ExamUiState = {
  checking: false,
  hasPreviousSubmission: false,
  score: null,
  percentile: null,
  submissionMessage: null,
  resultError: null,
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
      };
    case 'CHECK_SUCCESS':
      return {
        checking: false,
        hasPreviousSubmission: true,
        score: action.payload.score,
        percentile: action.payload.percentile,
        submissionMessage: action.payload.message,
        resultError: null,
      };
    case 'CHECK_ERROR':
      return {
        checking: false,
        hasPreviousSubmission: false,
        score: null,
        percentile: null,
        submissionMessage: null,
        resultError: action.payload ?? null,
      };
    case 'SUBMIT_SUCCESS':
      return {
        checking: false,
        hasPreviousSubmission: true,
        score: action.payload.score,
        percentile: action.payload.percentile,
        submissionMessage: action.payload.message,
        resultError: null,
      };
    case 'SUBMIT_ERROR':
      return {
        ...state,
        checking: false,
        hasPreviousSubmission: false,
        score: null,
        percentile: null,
        submissionMessage: null,
        resultError: action.payload ?? null,
      };
    default:
      return state;
  }
};

export function useExamUi(initialState: ExamUiState = examUiInitialState) {
  return useReducer(examUiReducer, initialState);
}
