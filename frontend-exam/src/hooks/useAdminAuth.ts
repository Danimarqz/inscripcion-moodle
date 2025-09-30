import { useEffect, useState } from 'preact/hooks';

import { validateAdminToken } from '../services/adminService';
import { getAuthToken, removeAuthToken, redirectToLogin } from '../utils/adminAuth';

interface UseAdminAuthOptions {
  redirectOnMissing?: boolean;
  validate?: boolean;
}

export function useAdminAuth(options: UseAdminAuthOptions = {}) {
  const { redirectOnMissing = true, validate = true } = options;
  const [token, setToken] = useState<string | null>(() => getAuthToken());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function ensureAuth() {
      const currentToken = getAuthToken();

      if (!currentToken) {
        if (redirectOnMissing) {
          redirectToLogin();
        }

        if (!cancelled) {
          setToken(null);
          setError(null);
          setLoading(false);
        }
        return;
      }

      if (!cancelled) {
        setToken(currentToken);
      }

      if (!validate) {
        if (!cancelled) {
          setError(null);
          setLoading(false);
        }
        return;
      }

      try {
        const isValid = await validateAdminToken(currentToken);
        if (!isValid) {
          removeAuthToken();
          if (redirectOnMissing) {
            redirectToLogin();
          }
          if (!cancelled) {
            setToken(null);
            setError('Sesión no válida');
          }
          return;
        }
        if (!cancelled) {
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    ensureAuth();

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
