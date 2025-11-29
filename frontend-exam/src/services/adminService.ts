import type {
  AdminSubmission,
  AdminSubmissionsResponse,
  Exam,
  ExamCreateWithQuestions,
  ExamEdit,
  ExamOfficialResult,
  AdminOfficialResultsResponse,
  CreateOfficialResultPayload,
  ImportOfficialResultsSummary,
  SubmissionUpdatePayload,
  SyncMoodleUsersResponse,
} from '../types/exam';

const API_URL = import.meta.env.PUBLIC_API_URL;

interface AdminLoginPayload {
  username: string;
  password: string;
}

interface TokenResponse {
  access_token: string;
  token_type: string;
}

export async function adminLogin(payload: AdminLoginPayload): Promise<TokenResponse> {
  const response = await fetch(`${API_URL}/admin/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.detail || 'Error de autenticacion');
  }

  return response.json();
}

export async function getAdminExams(token: string): Promise<Exam[]> {
  const response = await fetch(`${API_URL}/admin/exams`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.detail || 'Error fetching exams');
  }

  return response.json();
}

export async function createExam(examData: ExamCreateWithQuestions, token: string): Promise<Exam> {
  const response = await fetch(`${API_URL}/admin/exams`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(examData),
  });
  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.detail || 'Error creating exam');
  }
  return response.json();
}

export async function editExam(examId: number, examData: ExamEdit, token: string): Promise<Exam> {
  const response = await fetch(`${API_URL}/admin/exams/${examId}/edit`, {
    method: 'PUT',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(examData),
  });
  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.detail || 'Error editing exam');
  }
  return response.json();
}

export async function deleteExam(examId: number, token: string): Promise<void> {
  const response = await fetch(`${API_URL}/admin/exams/${examId}/delete`, {
    method: 'DELETE',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.detail || 'Error deleting exam');
  }
}

export async function validateAdminToken(token: string): Promise<boolean> {
  const response = await fetch(`${API_URL}/admin/check-token`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  return response.ok;
}

export async function getExamById(examId: number, token: string): Promise<ExamEdit> {
  const response = await fetch(`${API_URL}/admin/exams/${examId}`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  if (!response.ok) {
    throw new Error('Error fetching exams');
  }
  return await response.json();
}

interface FetchSubmissionsOptions {
  limit?: number;
  offset?: number;
  firstLoad?: boolean;
  search?: string;
  orderBy?: 'submitted_at' | 'score' | 'name' | 'surname';
  orderDir?: 'asc' | 'desc';
  moodleSynced?: boolean;
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

export async function getExamSubmissions(
  examId: number,
  token: string,
  options: FetchSubmissionsOptions = {},
): Promise<AdminSubmissionsResponse> {
  const params = new URLSearchParams({ exam_id: String(examId) });
  if (options.limit !== undefined) {
    params.set('limit', String(options.limit));
  }
  if (options.offset !== undefined) {
    params.set('offset', String(options.offset));
  }
  if (options.search) {
    params.set('search', options.search);
  }
  if (options.orderBy) {
    params.set('order_by', options.orderBy);
  }
  if (options.orderDir) {
    params.set('order_dir', options.orderDir);
  }
  if (options.moodleSynced !== undefined) {
    params.set('moodle_synced', String(options.moodleSynced));
  }
  const firstLoad = options.firstLoad ?? false;
  params.set('first_load', String(firstLoad));
  const response = await fetch(`${API_URL}/admin/results?${params.toString()}`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.detail || 'Error fetching submissions');
  }

  return response.json();
}

export async function fetchSubmissionEmailList(
  examId: number,
  token: string,
  options: FetchSubmissionsOptions = {},
): Promise<string[]> {
  const params = new URLSearchParams({ exam_id: String(examId) });
  if (options.search) {
    params.set('search', options.search);
  }
  if (options.orderBy) {
    params.set('order_by', options.orderBy);
  }
  if (options.orderDir) {
    params.set('order_dir', options.orderDir);
  }
  if (options.moodleSynced !== undefined) {
    params.set('moodle_synced', String(options.moodleSynced));
  }
  const response = await fetch(`${API_URL}/admin/results/emails/list?${params.toString()}`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.detail || 'Error fetching submission emails');
  }
  return response.json();
}


export async function downloadSubmissionEmails(
  examId: number,
  token: string,
  options: FetchSubmissionsOptions = {},
): Promise<string> {
  const params = new URLSearchParams({ exam_id: String(examId) });
  if (options.search) {
    params.set('search', options.search);
  }
  if (options.orderBy) {
    params.set('order_by', options.orderBy);
  }
  if (options.orderDir) {
    params.set('order_dir', options.orderDir);
  }
  if (options.moodleSynced !== undefined) {
    params.set('moodle_synced', String(options.moodleSynced));
  }
  const response = await fetch(`${API_URL}/admin/results/emails?${params.toString()}`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.detail || 'Error downloading submission emails');
  }

  return response.text();
}

export async function sendSubmissionEmails(
  payload: SendSubmissionEmailsPayload,
  token: string,
): Promise<{ sent: number }> {
  const response = await fetch(`${API_URL}/admin/results/emails/send`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.detail || 'Error sending emails');
  }

  return response.json();
}

export async function syncMoodleUsers(token: string): Promise<SyncMoodleUsersResponse> {
  const response = await fetch(`${API_URL}/admin/moodle/sync-users`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.detail || 'Error synchronizing Moodle users');
  }
  return response.json();
}

interface FetchOfficialResultsOptions {
  limit?: number;
  offset?: number;
  orderBy?: 'dni' | 'nombre' | 'apellidos' | 'usuario' | 'creado';
  orderDir?: 'asc' | 'desc';
}

export async function getOfficialResults(
  examId: number,
  token: string,
  options: FetchOfficialResultsOptions = {},
): Promise<AdminOfficialResultsResponse> {
  const params = new URLSearchParams();
  if (options.limit !== undefined) {
    params.set('limit', String(options.limit));
  }
  if (options.offset !== undefined) {
    params.set('offset', String(options.offset));
  }
  if (options.orderBy) {
    params.set('order_by', options.orderBy);
  }
  if (options.orderDir) {
    params.set('order_dir', options.orderDir);
  }

  const query = params.toString();
  const response = await fetch(`${API_URL}/admin/exams/${examId}/results/official${query ? `?${query}` : ''}`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.detail || 'Error fetching official results');
  }

  return response.json();
}

export async function importOfficialResults(
  examId: number,
  file: File,
  token: string,
  replaceExisting = true,
): Promise<ImportOfficialResultsSummary> {
  const formData = new FormData();
  formData.append('file', file);

  const response = await fetch(
    `${API_URL}/admin/exams/${examId}/results/import?replace_existing=${replaceExisting}`,
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
      },
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
  const response = await fetch(`${API_URL}/admin/exams/${examId}/results/official`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.detail || 'Error creating official result');
  }

  return response.json();
}

export async function updateSubmissionAttempt(
  submissionId: number,
  payload: SubmissionUpdatePayload,
  token: string,
): Promise<AdminSubmission> {
  const response = await fetch(`${API_URL}/admin/results/${submissionId}`, {
    method: 'PUT',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.detail || 'Error updating submission');
  }

  return response.json();
}

export async function deleteSubmissionAttempt(submissionId: number, token: string): Promise<void> {
  const response = await fetch(`${API_URL}/admin/results/${submissionId}`, {
    method: 'DELETE',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.detail || 'Error deleting submission');
  }
}
