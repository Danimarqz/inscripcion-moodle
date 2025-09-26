export function getAuthToken(): string | null {
  return localStorage.getItem('admin_access_token');
}

export function saveAuthToken(token: string): void {
  localStorage.setItem('admin_access_token', token);
}

export function removeAuthToken(): void {
  localStorage.removeItem('admin_access_token');
}

export function redirectToLogin(): void {
  window.location.href = '/admin/login';
}
