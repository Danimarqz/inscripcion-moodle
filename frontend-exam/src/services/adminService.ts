import type { Exam } from '../types/exam';

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

export async function getExamsAdmin(token: string): Promise<Exam[]> {
  const response = await fetch(`${API_URL}/exams`);

  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.detail || 'Error al obtener exámenes');
  }

  return response.json();
}

export function saveAuthToken(token: string) {
  localStorage.setItem('admin_access_token', token);
}

export function getAuthToken(): string | null {
  return localStorage.getItem('admin_access_token');
}

export function removeAuthToken() {
  localStorage.removeItem('admin_access_token');
}
