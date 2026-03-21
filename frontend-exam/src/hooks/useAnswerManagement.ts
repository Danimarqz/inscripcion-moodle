import { useCallback, useMemo, useState } from 'preact/hooks';
import type { Question } from '../types/exam';

export function useAnswerManagement(questions: Question[]) {
  const [userAnswers, setUserAnswers] = useState<Record<number, string>>({});

  const questionEntries = useMemo(
    () => questions.map((question, index) => ({ index, question })),
    [questions],
  );
  const activeEntries = useMemo(
    () => questionEntries.filter(({ question }) => question.is_active !== false),
    [questionEntries],
  );
  const reserveEntries = useMemo(
    () => questionEntries.filter(({ question }) => question.is_active === false),
    [questionEntries],
  );

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

  return { userAnswers, setAnswer, clearAnswer, activeEntries, reserveEntries, resetAnswers };
}
