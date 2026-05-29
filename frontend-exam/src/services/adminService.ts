
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

const DOWNLOAD_TIMEOUT_MS = 30_000;

async function fetchWithTimeout(
  url: string,
  init: RequestInit,
  timeoutMs: number,
): Promise<Response> {
  const controller = new AbortController();
  const timeoutId = setTimeout(
    () => controller.abort(new DOMException('Request timed out', 'TimeoutError')),
    timeoutMs,
  );
  try {
    return await fetch(url, { ...init, signal: controller.signal });
  } finally {
    clearTimeout(timeoutId);
  }
}

export async function downloadSubmissionEmails(
  examId: number,
  token: string,
  options: FetchSubmissionsOptions = {},
): Promise<string> {
  const params = buildParams(examId, options);
  const API_URL = import.meta.env.PUBLIC_API_URL;
  const response = await fetchWithTimeout(
    `${API_URL}/admin/results/emails?${params.toString()}`,
    { headers: { Authorization: `Bearer ${token}` } },
    DOWNLOAD_TIMEOUT_MS,
  );
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
  const response = await fetchWithTimeout(
    `${API_URL}/admin/exams/${examId}/results/analysis?${params.toString()}`,
    { headers: { Authorization: `Bearer ${token}` } },
    DOWNLOAD_TIMEOUT_MS,
  );
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

type SseEvent = {
  status: string;
  message?: string;
  [key: string]: unknown;
};

async function readSseStream(
  response: Response,
  onProgress: (message: string) => void,
): Promise<SseEvent> {
  if (!response.body) throw new Error('No response body');
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() ?? '';
    for (const line of lines) {
      if (!line.startsWith('data: ')) continue;
      const data = JSON.parse(line.slice(6)) as SseEvent;
      if (data.status === 'error') throw new Error(data.message ?? 'Error durante la sincronización');
      if (data.status === 'done') return data;
      if (data.message) onProgress(data.message);
    }
  }
  throw new Error('La conexión se cerró antes de completar la sincronización');
}

export async function syncMoodleUsersStream(
  token: string,
  onProgress: (message: string) => void,
): Promise<{ checked: number; synced: number; failed: number }> {
  const API_URL = import.meta.env.PUBLIC_API_URL;
  const response = await fetch(`${API_URL}/admin/moodle/sync-users`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error('Error al iniciar la sincronización');
  const data = await readSseStream(response, onProgress);
  return {
    checked: (data.checked as number) ?? 0,
    synced: (data.synced as number) ?? 0,
    failed: (data.failed as number) ?? 0,
  };
}

export async function syncOfficialResultsMoodleStream(
  examId: number,
  token: string,
  onProgress: (message: string) => void,
): Promise<{ matched: number }> {
  const API_URL = import.meta.env.PUBLIC_API_URL;
  const response = await fetch(`${API_URL}/admin/exams/${examId}/results/official/sync-moodle`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error('Error al iniciar la sincronización con Moodle');
  const data = await readSseStream(response, onProgress);
  return { matched: (data.matched as number) ?? 0 };
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
    const text = await response.text().catch(() => '');
    let message = 'Error importing official results';
    try {
      const json = JSON.parse(text) as Record<string, unknown>;
      if (typeof json.detail === 'string') message = json.detail;
    } catch {
      if (text.trim()) message = text.trim();
    }
    throw new Error(message);
  }

  return response.json() as Promise<ImportOfficialResultsSummary>;
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

export async function downloadOfficialResultsTemplate(examId: number, token: string): Promise<void> {
  const API_URL = import.meta.env.PUBLIC_API_URL;
  const response = await fetch(
    `${API_URL}/admin/exams/${examId}/results/official/template`,
    {
      method: 'GET',
      headers: { Authorization: `Bearer ${token}` },
    },
  );
  if (!response.ok) {
    throw new Error('Error descargando plantilla');
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = 'plantilla_resultados_oficiales.xlsx';
  a.click();
  URL.revokeObjectURL(url);
}

export async function getQuestionFeedbackVideo(
  questionId: number,
  token: string,
): Promise<string | null> {
  const res = await request<{ id: number; feedback_video_key: string | null }>(
    `/admin/questions/${questionId}`,
    { method: 'GET', token },
  );
  return res.feedback_video_key ?? null;
}

export async function patchQuestionFeedbackVideo(
  questionId: number,
  key: string | null,
  token: string,
): Promise<void> {
  return request<void>(`/admin/questions/${questionId}`, {
    method: 'PATCH',
    token,
    body: JSON.stringify({ feedback_video_key: key }),
  });
}
