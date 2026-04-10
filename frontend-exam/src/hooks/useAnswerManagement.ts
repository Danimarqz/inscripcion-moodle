import { useCallback, useMemo, useState } from 'preact/hooks';
import type { Question } from '../types/exam';

export function useAnswerManagement(questions: Question[]) {
  const [userAnswers, setUserAnswers] = useState<Record<number, string>>({});

  const entries = useMemo(() => {
    const mapped = questions.map((question, index) => ({ index, question }));
    mapped.sort((a, b) => {
      const aName = typeof a.question.name === 'number' ? a.question.name : Number.POSITIVE_INFINITY;
      const bName = typeof b.question.name === 'number' ? b.question.name : Number.POSITIVE_INFINITY;
      if (aName !== bName) return aName - bName;
      return a.index - b.index;
    });
    return mapped;
  }, [questions]);

  const setAnswer = useCallback((questionId: number, option: string) => {
    const normalizedOption = option.toUpperCase();
    setUserAnswers((prev) => ({
      ...prev,
      [questionId]: normalizedOption,
    }));
  }, []);

  const clearAnswer = useCallback((questionId: number) => {
    setUserAnswers((prev) => {
      if (!(questionId in prev)) {
        return prev;
      }
      const next = { ...prev };
      delete next[questionId];
      return next;
    });
  }, []);

  const resetAnswers = useCallback(() => {
    setUserAnswers({});
  }, []);

  return { userAnswers, setAnswer, clearAnswer, entries, resetAnswers };
}
