import { useEffect, useState } from 'preact/hooks';

import { validateAdminToken } from '../services/adminService';
import { getAuthToken, removeAuthToken, redirectToLogin } from '../utils/adminAuth';

export function useAdminAuth() {
  const [token, setToken] = useState<string | null>(() => getAuthToken());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function validate() {
      const currentToken = getAuthToken();
      if (!currentToken) {
        redirectToLogin();
        if (!cancelled) {
          setToken(null);
          setLoading(false);
        }
        return;
      }

      setToken(currentToken);

      try {
        const isValid = await validateAdminToken(currentToken);
        if (!isValid) {
          removeAuthToken();
          redirectToLogin();
          if (!cancelled) {
            setToken(null);
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

    validate();

    return () => {
      cancelled = true;
    };
  }, []);

  return {
    token,
    loading,
    error,
  } as const;
}
