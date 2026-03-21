const STORAGE_KEY = 'admin_access_token';
const LOGIN_ROUTE = '/admin/login';

function getStorage(): Storage | null {
  if (typeof window === 'undefined' || typeof window.localStorage === 'undefined') {
    return null;
  }

  try {
    return window.localStorage;
  } catch (error) {
    return null;
  }
}

function isTokenExpired(token: string): boolean {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return true;
    const payload = JSON.parse(atob(parts[1]));
    if (typeof payload.exp !== 'number') return true;
    return Date.now() >= payload.exp * 1000;
  } catch {
    return true;
  }
}

export function getAuthToken(): string | null {
  const storage = getStorage();
  const token = storage ? storage.getItem(STORAGE_KEY) : null;
  if (token && isTokenExpired(token)) {
    removeAuthToken();
    return null;
  }
  return token;
}

export function saveAuthToken(token: string): void {
  const storage = getStorage();
  storage?.setItem(STORAGE_KEY, token);
}

export function removeAuthToken(): void {
  const storage = getStorage();
  storage?.removeItem(STORAGE_KEY);
}

export function redirectToLogin(): void {
  if (typeof window !== 'undefined') {
    window.location.href = LOGIN_ROUTE;
  }
}
