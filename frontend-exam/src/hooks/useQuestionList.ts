import { useCallback, useState } from 'preact/hooks';

type QuestionShape = {
  correct_option: string;
  text?: string;
  id?: number;
  is_active?: boolean;
  is_cancelled?: boolean;
  name?: number;
};

type UpdateField = keyof QuestionShape;

const DEFAULT_QUESTION_STATE = { correct_option: 'A', is_active: true, is_cancelled: false };

export function useQuestionList<T extends QuestionShape>(
  initial: T[] = [{ ...DEFAULT_QUESTION_STATE } as T],
) {
  const [questions, setQuestions] = useState<T[]>(initial);

  const setAll = useCallback((next: T[]) => {
    setQuestions(
      next.length
        ? next
        : ([{ ...DEFAULT_QUESTION_STATE } as T]),
    );
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
    setQuestions((prev) => [...prev, { ...DEFAULT_QUESTION_STATE } as T]);
  }, []);

  const removeQuestion = useCallback((index: number) => {
    setQuestions((prev) => prev.filter((_, i) => i !== index));
  }, []);

  return { questions, setAll, updateQuestion, addQuestion, removeQuestion } as const;
}
