import type { Exam, ExamCreateWithQuestions, ExamEdit } from '../types/exam';

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
    throw new Error(errorData.detail || 'Error de autenticación');
  }

  return response.json();
}
export async function createExam(examData: ExamCreateWithQuestions, token: string): Promise<Exam> {
  console.log(JSON.stringify(examData, null, 2));
  const response = await fetch(`${API_URL}/admin/exams`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(examData)
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
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(examData)
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
      'Authorization': `Bearer ${token}`,
    }
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
      'Authorization': `Bearer ${token}`,
    }
  });
  if (!response.ok) {
    throw new Error('Error fetching exams');
  }
  return await response.json();
}