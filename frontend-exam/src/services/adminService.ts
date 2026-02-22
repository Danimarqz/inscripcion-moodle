
import type {
  AdminSubmission,
  AdminSubmissionsResponse,
  Exam,
  ExamCreateWithQuestions,
  ExamEdit,
  ExamOfficialResult,
  AdminOfficialResultsResponse,
  CreateOfficialResultPayload,
  EditOfficialResultPayload,
  ImportOfficialResultsSummary,
  SubmissionUpdatePayload,
  SyncMoodleUsersResponse,
} from '../types/exam';
import { request } from './api';

interface AdminLoginPayload {
  username: string;
  password: string;
}

interface TokenResponse {
  access_token: string;
  token_type: string;
}

export async function adminLogin(payload: AdminLoginPayload): Promise<TokenResponse> {
  return request<TokenResponse>('/admin/login', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function getAdminExams(token: string): Promise<Exam[]> {
  return request<Exam[]>('/admin/exams', {
    method: 'GET',
    token,
  });
}

export async function createExam(examData: ExamCreateWithQuestions, token: string): Promise<Exam> {
  return request<Exam>('/admin/exams', {
    method: 'POST',
    token,
    body: JSON.stringify(examData),
  });
}

export async function editExam(examId: number, examData: ExamEdit, token: string): Promise<Exam> {
  return request<Exam>(`/admin/exams/${examId}`, {
    method: 'PUT',
    token,
    body: JSON.stringify(examData),
  });
}

export async function deleteExam(examId: number, token: string): Promise<void> {
  return request<void>(`/admin/exams/${examId}/delete`, {
    method: 'DELETE',
    token,
  });
}

export async function validateAdminToken(token: string): Promise<boolean> {
  try {
    await request('/admin/check-token', { method: 'GET', token });
    return true;
  } catch {
    return false;
  }
}

export async function getExamById(examId: number, token: string): Promise<ExamEdit> {
  return request<ExamEdit>(`/admin/exams/${examId}`, {
    method: 'GET',
    token,
  });
}

interface FetchSubmissionsOptions {
  limit?: number;
  offset?: number;
  firstLoad?: boolean;
  search?: string;
  orderBy?: 'submitted_at' | 'score' | 'name' | 'surname';
  orderDir?: 'asc' | 'desc';
  moodleSynced?: boolean;
  type?: string;
}

export interface SubmissionEmailAttachmentPayload {
  filename: string;
  content: string;
  content_type: string;
}

export interface SendSubmissionEmailsPayload {
  exam_id: number;
  subject: string;
  body: string;
  recipients: string[];
  search?: string;
  order_by?: string;
  order_dir?: string;
  moodle_synced?: boolean;
  attachments?: SubmissionEmailAttachmentPayload[];
}

function buildParams(examId: number, options: FetchSubmissionsOptions): URLSearchParams {
  const params = new URLSearchParams({ exam_id: String(examId) });
  if (options.limit !== undefined) params.set('limit', String(options.limit));
  if (options.offset !== undefined) params.set('offset', String(options.offset));
  if (options.search) params.set('search', options.search);
  if (options.orderBy) params.set('order_by', options.orderBy);
  if (options.orderDir) params.set('order_dir', options.orderDir);
  if (options?.moodleSynced !== undefined) {
    params.append('moodle_synced', String(options.moodleSynced));
  }
  if (options?.type) {
    params.append('type', options.type);
  }
  return params;
}

export async function getExamSubmissions(
  examId: number,
  token: string,
  options: FetchSubmissionsOptions = {},
): Promise<AdminSubmissionsResponse> {
  const params = buildParams(examId, options);
  params.set('first_load', String(options.firstLoad ?? false));
  return request<AdminSubmissionsResponse>(`/admin/results?${params.toString()}`, {
    method: 'GET',
    token,
  });
}

export async function fetchSubmissionEmailList(
  examId: number,
  token: string,
  options: FetchSubmissionsOptions = {},
): Promise<string[]> {
  const params = buildParams(examId, options);
  return request<string[]>(`/admin/results/emails/list?${params.toString()}`, {
    method: 'GET',
    token,
  });
}

export async function downloadSubmissionEmails(
  examId: number,
  token: string,
  options: FetchSubmissionsOptions = {},
): Promise<string> {
  // Use text() response manually if request helper assumes json
  const params = buildParams(examId, options);
  const API_URL = import.meta.env.PUBLIC_API_URL;
  const response = await fetch(`${API_URL}/admin/results/emails?${params.toString()}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error('Error downloading submission emails');
  return response.text();
}

export async function downloadSubmissionsAnalysis(
  examId: number,
  token: string,
  options: FetchSubmissionsOptions = {},
): Promise<Blob> {
  const params = buildParams(examId, options);
  const API_URL = import.meta.env.PUBLIC_API_URL;
  const response = await fetch(`${API_URL}/admin/exams/${examId}/results/analysis?${params.toString()}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error('Error downloading analysis');
  return response.blob();
}

export async function sendSubmissionEmails(
  payload: SendSubmissionEmailsPayload,
  token: string,
): Promise<{ sent: number }> {
  return request<{ sent: number }>('/admin/results/emails/send', {
    method: 'POST',
    token,
    body: JSON.stringify(payload),
  });
}

export async function syncMoodleUsers(token: string): Promise<SyncMoodleUsersResponse> {
  return request<SyncMoodleUsersResponse>('/admin/moodle/sync-users', {
    method: 'POST',
    token,
  });
}

export async function syncOfficialResultsMoodle(
  examId: number,
  token: string,
): Promise<{ matched: number; synced: number }> {
  return request<{ matched: number; synced: number }>(
    `/admin/exams/${examId}/results/official/sync-moodle`,
    {
      method: 'POST',
      token,
    },
  );
}

export type FetchOfficialResultsOptions = {
  limit: number;
  offset: number;
  orderBy: 'dni' | 'nombre' | 'apellidos' | 'usuario' | 'creado';
  orderDir: 'asc' | 'desc';
  type?: string;
  search?: string;
  hasUser?: boolean;
};

export async function getOfficialResults(
  examId: number,
  token: string,
  options: FetchOfficialResultsOptions,
): Promise<AdminOfficialResultsResponse> {
  const params = new URLSearchParams({
    limit: options.limit.toString(),
    offset: options.offset.toString(),
    order_by: options.orderBy,
    order_dir: options.orderDir,
  });
  if (options.type) {
    params.append('type', options.type);
  }
  if (options.search) {
    params.append('search', options.search);
  }
  if (options.hasUser !== undefined) {
    params.append('has_user', String(options.hasUser));
  }

  const query = params.toString();
  return request<AdminOfficialResultsResponse>(`/admin/exams/${examId}/results/official${query ? `?${query}` : ''}`, {
    method: 'GET',
    token,
  });
}

export async function importOfficialResults(
  examId: number,
  file: File,
  token: string,
  replaceExisting = true,
): Promise<ImportOfficialResultsSummary> {
  const formData = new FormData();
  formData.append('file', file);

  // FormData handling is special, fetch handles content-type boundary
  const API_URL = import.meta.env.PUBLIC_API_URL;
  const response = await fetch(
    `${API_URL}/admin/exams/${examId}/results/import?replace_existing=${replaceExisting}`,
    {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body: formData,
    },
  );

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.detail || 'Error importing official results');
  }

  return response.json();
}

export async function createOfficialResult(
  examId: number,
  payload: CreateOfficialResultPayload,
  token: string,
): Promise<ExamOfficialResult> {
  return request<ExamOfficialResult>(`/admin/exams/${examId}/results/official`, {
    method: 'POST',
    token,
    body: JSON.stringify(payload),
  });
}

export async function updateOfficialResult(
  id: number,
  payload: EditOfficialResultPayload,
  token: string,
): Promise<ExamOfficialResult> {
  return request<ExamOfficialResult>(`/admin/exams/results/official/${id}`, {
    method: 'PUT',
    token,
    body: JSON.stringify(payload),
  });
}

export async function deleteOfficialResult(id: number, token: string): Promise<void> {
  return request<void>(`/admin/exams/results/official/${id}`, {
    method: 'DELETE',
    token,
  });
}

export async function getSubmission(
  submissionId: number,
  token: string,
): Promise<AdminSubmission> {
  return request<AdminSubmission>(`/admin/results/${submissionId}`, {
    method: 'GET',
    token,
  });
}

export async function updateSubmissionAttempt(
  submissionId: number,
  payload: SubmissionUpdatePayload,
  token: string,
): Promise<AdminSubmission> {
  return request<AdminSubmission>(`/admin/results/${submissionId}`, {
    method: 'PUT',
    token,
    body: JSON.stringify(payload),
  });
}

export async function deleteSubmissionAttempt(submissionId: number, token: string): Promise<void> {
  return request<void>(`/admin/results/${submissionId}`, {
    method: 'DELETE',
    token,
  });
}
