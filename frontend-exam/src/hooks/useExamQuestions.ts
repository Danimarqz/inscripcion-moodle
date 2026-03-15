import { useEffect, useState } from 'preact/hooks';

import { getQuestions } from '../services/examService';
import type { Question } from '../types/exam';

export function useExamQuestions(examId: number) {
  const [questions, setQuestions] = useState<Question[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function fetchQuestions() {
      setLoading(true);
      setError(null);
      try {
        const data = await getQuestions(examId);
        if (!cancelled) {
          setQuestions(data);
        }
      } catch (fetchError: unknown) {
        if (!cancelled) {
          setError(fetchError instanceof Error ? fetchError.message : String(fetchError));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchQuestions();

    return () => {
      cancelled = true;
    };
  }, [examId]);

  return { questions, loading, error };
}
