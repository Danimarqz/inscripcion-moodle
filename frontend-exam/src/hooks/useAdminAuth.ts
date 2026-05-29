import { useEffect, useState } from 'preact/hooks';

import { checkAdminSession } from '../services/adminService';
import { getAuthToken, saveAuthToken, removeAuthToken, redirectToLogin, SESSION_MARKER } from '../utils/adminAuth';

interface UseAdminAuthOptions {
  redirectOnMissing?: boolean;
  // When false, skip the server session check entirely and just report the
  // in-memory marker (cheap, no network — used by the global Header on public
  // pages). The admin pages themselves are also gated server-side by the
  // /admin/* SSR middleware.
  validate?: boolean;
}

export function useAdminAuth(options: UseAdminAuthOptions = {}) {
  const { redirectOnMissing = true, validate = true } = options;
  // `token` is now a client-side session marker (the real JWT lives in the
  // httpOnly cookie). It stays in the return value so existing consumers that
  // thread `token` keep working; it is never sent to the API.
  const [token, setToken] = useState<string | null>(() => getAuthToken());
  const [loading, setLoading] = useState(validate);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    if (!validate) {
      // Client-only: no network. We can't read the httpOnly cookie, so this
      // reflects only an in-memory marker set earlier this session.
      setToken(getAuthToken());
      setLoading(false);
      return;
    }

    async function ensureAuth() {
      // Source of truth is the cookie: ask the backend whether the session is
      // valid (credentials are sent automatically by request()).
      const valid = await checkAdminSession();
      if (cancelled) return;

      if (!valid) {
        removeAuthToken();
        setToken(null);
        if (redirectOnMissing) {
          redirectToLogin();
        } else {
          setError('Sesión no válida');
        }
        setLoading(false);
        return;
      }

      // Authenticated. Ensure a truthy marker exists for client-side guards.
      if (!getAuthToken()) {
        saveAuthToken(SESSION_MARKER);
      }
      setToken(getAuthToken());
      setError(null);
      setLoading(false);
    }

    void ensureAuth();

    return () => {
      cancelled = true;
    };
  }, [redirectOnMissing, validate]);

  return {
    token,
    loading,
    error,
  } as const;
}
