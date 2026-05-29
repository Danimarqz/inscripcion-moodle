const LOGIN_ROUTE = '/admin/login';

// Sentinel returned after a reload, when the real JWT lives only in the
// httpOnly cookie (unreadable by JS). It is a truthy "we have a session" marker
// for client-side guards; it is NEVER sent to the API (see api.ts).
export const SESSION_MARKER = 'cookie-session';

// Admin auth lives in an httpOnly cookie set by the backend. We keep an
// IN-MEMORY mirror only so existing client-side `token` guards keep working
// within a session; nothing is persisted to localStorage anymore.
let sessionToken: string | null = null;

function isTokenExpired(token: string): boolean {
  // The session marker has no exp; treat it as live (the cookie/server decides).
  if (token === SESSION_MARKER) return false;
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
  if (sessionToken && isTokenExpired(sessionToken)) {
    sessionToken = null;
  }
  return sessionToken;
}

export function saveAuthToken(token: string): void {
  sessionToken = token;
}

export function removeAuthToken(): void {
  sessionToken = null;
}

export function redirectToLogin(): void {
  if (typeof window !== 'undefined') {
    window.location.href = LOGIN_ROUTE;
  }
}
