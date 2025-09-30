import { useEffect, useState } from 'preact/hooks';

import { getAdminExams } from '../services/adminService';
import type { Exam } from '../types/exam';

export function useAdminExams(token: string | null, enabled: boolean) {
  const [exams, setExams] = useState<Exam[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function fetchData(currentToken: string) {
      setLoading(true);
      setError(null);
      try {
        const examList = await getAdminExams(currentToken);
        if (!cancelled) {
          setExams(examList);
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

    if (enabled && token) {
      void fetchData(token);
    }

    return () => {
      cancelled = true;
    };
  }, [enabled, token]);

  return { exams, loading, error } as const;
}
