import type {
  Exam,
  ExamOut,
  ExamSubmissionPayload,
  Question,
  OfficialResultCheckPayload,
  OfficialResultCheckResponse,
  UserSubmissionCheck,
} from '../types/exam';
import { request } from './api';

export async function getExams(): Promise<Exam[]> {
  return request<Exam[]>('/exams', { method: 'GET' });
}

export async function getExamBySlug(slug: string): Promise<Exam> {
  return request<Exam>(`/exams/slug/${slug}`, { method: 'GET' });
}

export async function getQuestions(examId: number): Promise<Question[]> {
  return request<Question[]>(`/exams/${examId}/questions`, { method: 'GET' });
}

export async function submitExam(payload: ExamSubmissionPayload): Promise<ExamOut> {
  if (!payload.eligibility_confirmed) {
    throw new Error('No se ha verificado que perteneces al examen oficial.');
  }

  return request<ExamOut>('/submit-exam', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function checkSubmission(payload: UserSubmissionCheck): Promise<ExamOut> {
  return request<ExamOut>('/check_submission', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function checkOfficialResult(
  payload: OfficialResultCheckPayload,
): Promise<OfficialResultCheckResponse> {
  return request<OfficialResultCheckResponse>('/check-official-result', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}
