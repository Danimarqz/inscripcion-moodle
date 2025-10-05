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

export function getAuthToken(): string | null {
  const storage = getStorage();
  return storage ? storage.getItem(STORAGE_KEY) : null;
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
