import { useCallback, useState } from 'preact/hooks';

type QuestionShape = { correct_option: string; text?: string; id?: number };

type UpdateField = keyof QuestionShape;

export function useQuestionList<T extends QuestionShape>(initial: T[] = [{ correct_option: 'A' } as T]) {
  const [questions, setQuestions] = useState<T[]>(initial);

  const setAll = useCallback((next: T[]) => {
    setQuestions(next.length ? next : ([{ correct_option: 'A' } as T]));
  }, []);

  const updateQuestion = useCallback((index: number, field: UpdateField, value: string) => {
    setQuestions((prev) => {
      const copy = [...prev];
      copy[index] = { ...copy[index], [field]: value } as T;
      return copy;
    });
  }, []);

  const addQuestion = useCallback(() => {
    setQuestions((prev) => [...prev, { correct_option: 'A' } as T]);
  }, []);

  const removeQuestion = useCallback((index: number) => {
    setQuestions((prev) => prev.filter((_, i) => i !== index));
  }, []);

  return { questions, setAll, updateQuestion, addQuestion, removeQuestion } as const;
}
