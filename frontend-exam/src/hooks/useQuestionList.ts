import { useCallback, useState } from 'preact/hooks';

type QuestionShape = { correct_option: string; text?: string; id?: number; is_active?: boolean };

type UpdateField = keyof QuestionShape;

export function useQuestionList<T extends QuestionShape>(
  initial: T[] = [{ correct_option: 'A', is_active: true } as T],
) {
  const [questions, setQuestions] = useState<T[]>(initial);

  const setAll = useCallback((next: T[]) => {
    setQuestions(next.length ? next : ([{ correct_option: 'A', is_active: true } as T]));
  }, []);

  const updateQuestion = useCallback(
    <K extends UpdateField>(index: number, field: K, value: QuestionShape[K]) => {
      setQuestions((prev) => {
        const copy = [...prev];
        copy[index] = { ...copy[index], [field]: value } as T;
        return copy;
      });
    },
    [],
  );

  const addQuestion = useCallback(() => {
    setQuestions((prev) => [...prev, { correct_option: 'A', is_active: true } as T]);
  }, []);

  const removeQuestion = useCallback((index: number) => {
    setQuestions((prev) => prev.filter((_, i) => i !== index));
  }, []);

  return { questions, setAll, updateQuestion, addQuestion, removeQuestion } as const;
}
