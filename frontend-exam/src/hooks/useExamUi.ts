import { useReducer, useCallback } from 'preact/hooks';
import type { ExamResultPayload, ExamUiState, ExamUiAction } from '../types/exam';

function examUiReducer(state: ExamUiState, action: ExamUiAction): ExamUiState {
  switch (action.type) {
    case 'RESET':
      return examUiInitialState;
    case 'CHECK_START':
      return { ...state, checking: true, resultError: null };
    case 'CHECK_SUCCESS':
    case 'SUBMIT_SUCCESS':
      return {
        checking: false,
        hasPreviousSubmission: true,
        resultError: null,
        ...payloadToUiState(action.payload),
      };
    case 'CHECK_ERROR':
      return {
        ...examUiInitialState,
        checking: false,
        resultError: action.payload ?? null,
      };
    case 'SUBMIT_ERROR':
      return {
        ...state,
        checking: false,
        resultError: action.payload ?? null,
      };
    default:
      return state;
  }
}

function payloadToUiState(payload: ExamResultPayload) {
  return {
    score: payload.score,
    percentile: payload.percentile,
    position: payload.position,
    totalSubmissions: payload.totalSubmissions,
    correctAnswers: payload.correctAnswers,
    incorrectAnswers: payload.incorrectAnswers,
    notAnswered: payload.notAnswered,
    totalQuestions: payload.totalQuestions,
    submissionMessage: payload.message,
    answersReview: payload.answersReview,
    maxScore: payload.max_score ?? null,
    secondaryMaxScores: payload.secondary_max_scores ?? null,
    isPassed: payload.isPassed,
    canEditMerits: payload.canEditMerits,
    allowMeritsEdit: payload.allowMeritsEdit,
    maxMerits: payload.maxMerits,
    merits: payload.merits,
    weightedScore: payload.weightedScore,
    examWeight: payload.examWeight,
    meritsPosition: payload.meritsPosition,
    meritsTotal: payload.meritsTotal,
    passedCount: payload.passedCount,
    groups: payload.groups,
  };
}

export function useExamUi() {
  const [state, dispatch] = useReducer(examUiReducer, examUiInitialState);

  const checkSuccess = useCallback((payload: ExamResultPayload) => {
    dispatch({ type: 'CHECK_SUCCESS', payload });
  }, []);

  const submitSuccess = useCallback((payload: ExamResultPayload) => {
    dispatch({ type: 'SUBMIT_SUCCESS', payload });
  }, []);

  const reset = useCallback(() => {
    dispatch({ type: 'RESET' });
  }, []);

  const checkStart = useCallback(() => {
    dispatch({ type: 'CHECK_START' });
  }, []);

  const checkError = useCallback((error?: string | null) => {
    dispatch({ type: 'CHECK_ERROR', payload: error });
  }, []);

  const submitError = useCallback((error?: string | null) => {
    dispatch({ type: 'SUBMIT_ERROR', payload: error });
  }, []);

  return { state, checkSuccess, submitSuccess, reset, checkStart, checkError, submitError };
}

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
  allowMeritsEdit: false,
  maxMerits: null,
  merits: null,
  weightedScore: null,
  examWeight: null,
  meritsPosition: null,
  meritsTotal: null,
  passedCount: null,
  groups: null,
};
