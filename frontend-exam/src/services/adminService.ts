import type {
  AdminSubmission,
  Exam,
  ExamCreateWithQuestions,
  ExamEdit,
  ExamOfficialResult,
  ImportOfficialResultsSummary,
  SubmissionUpdatePayload,
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

export async function getExamSubmissions(examId: number, token: string): Promise<AdminSubmission[]> {
  const response = await fetch(`${API_URL}/admin/results?exam_id=${examId}`, {
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

export async function getOfficialResults(examId: number, token: string): Promise<ExamOfficialResult[]> {
  const response = await fetch(`${API_URL}/admin/exams/${examId}/results/official`, {
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
