import { useEffect, useState } from 'preact/hooks';
import type {
  ExamCreateWithQuestions,
  ExamEdit,
  QuestionCreate,
  QuestionEdit,
} from '../types/exam';
import { createExam, editExam, getExamById } from '../services/adminService';
import { useAdminAuth } from '../hooks/useAdminAuth';
import { useQuestionList } from '../hooks/useQuestionList';
import { useAsyncTask } from '../hooks/useAsyncTask';

interface ExamFormProps {
  examId?: number;
}

export default function ExamForm({ examId }: ExamFormProps) {
  const { token, loading: authenticating, error: authError } = useAdminAuth();
  const { loading, error, run, setError } = useAsyncTask();
  const { questions, setAll, updateQuestion, addQuestion, removeQuestion } = useQuestionList<QuestionCreate | QuestionEdit>([
    { correct_option: 'A' },
  ]);

  const [examToEdit, setExamToEdit] = useState<ExamEdit | null>(null);
  const [name, setName] = useState('');
  const [isActive, setIsActive] = useState(false);
  const [showResponse, setShowResponse] = useState(false);

  useEffect(() => {
    if (authenticating) return;

    if (!token) {
      setError('No autorizado: token no disponible.');
      return;
    }

    if (!examId) {
      setExamToEdit(null);
      setName('');
      setIsActive(false);
      setShowResponse(false);
      setAll([{ correct_option: 'A' } as QuestionCreate]);
      setError(null);
      return;
    }

    void run(async () => {
      const examData = await getExamById(examId, token);
      setExamToEdit(examData);
      setName(examData.name ?? '');
      setIsActive(Boolean(examData.is_active));
      setShowResponse(Boolean(examData.show_response));
      setAll((examData.questions.length ? examData.questions : [{ correct_option: 'A' } as QuestionEdit]) as (
        QuestionCreate | QuestionEdit
      )[]);
    }).catch(() => undefined);
  }, [authenticating, examId, run, setAll, setError, token]);

  async function handleSubmit(event: Event) {
    event.preventDefault();
    setError(null);

    const trimmedName = name.trim();
    if (!trimmedName) {
      setError('El nombre del examen es obligatorio.');
      return;
    }
    if (questions.length === 0) {
      setError('Debe haber al menos una pregunta.');
      return;
    }
    if (questions.some((q) => !['A', 'B', 'C', 'D'].includes(q.correct_option.toUpperCase()))) {
      setError('Todas las preguntas deben tener una opción correcta válida (A, B, C o D).');
      return;
    }

    if (!token) {
      setError('No autorizado: token no disponible.');
      return;
    }

    const body: ExamCreateWithQuestions | ExamEdit = {
      name: trimmedName,
      is_active: isActive,
      show_response: showResponse,
      questions: questions.map((q) => ({
        id: 'id' in q ? q.id : undefined,
        correct_option: q.correct_option.toUpperCase(),
      })),
    };

    try {
      await run(async () => {
        if (examToEdit && examId) {
          await editExam(examId, body, token);
        } else {
          await createExam(body as ExamCreateWithQuestions, token);
        }
      });
      window.location.href = '/admin/dashboard';
    } catch (err) {
      // Error already handled by run(); nothing else to do.
    }
  }

  const isBusy = loading || authenticating;
  const formError = authError || error;

  return (
    <form onSubmit={handleSubmit} className="max-w-3xl mx-auto text-white font-sans">
      <h2 className="text-3xl font-bold text-center mb-8">{examToEdit ? 'Editar Examen' : 'Crear Examen'}</h2>

      {formError && (
        <p className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mb-6">{formError}</p>
      )}

      <div className="mb-6">
        <label className="block font-bold text-purple-500 mb-2">
          Nombre del examen:
          <input
            type="text"
            value={name}
            onInput={(e) => setName((e.target as HTMLInputElement).value)}
            required
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-purple-400 focus:ring-2 focus:ring-purple-400/50"
            disabled={isBusy}
          />
        </label>
      </div>

      <div className="mb-6 flex items-center">
        <input
          type="checkbox"
          checked={isActive}
          onChange={(e) => setIsActive(e.currentTarget.checked)}
          className="mr-2"
          disabled={isBusy}
        />
        <label className="font-bold text-purple-500">Activo</label>
      </div>

      <div className="mb-6 flex items-center">
        <input
          type="checkbox"
          checked={showResponse}
          onChange={(e) => setShowResponse(e.currentTarget.checked)}
          className="mr-2"
          disabled={isBusy}
        />
        <label className="font-bold text-purple-500">Mostrar respuestas</label>
      </div>

      <fieldset className="border border-[#444] p-4 rounded-lg" disabled={isBusy}>
        <legend className="font-bold text-xl mb-4">Preguntas</legend>
        {questions.map((q, i) => (
          <div key={i} className="bg-[#2a2d33] p-6 mb-4 rounded-lg shadow-lg">
            <label className="block font-bold text-purple-500 mb-2">
              Opción correcta:
              <select
                value={q.correct_option.toUpperCase()}
                onChange={(e) => updateQuestion(i, 'correct_option', e.currentTarget.value)}
                className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-purple-400 focus:ring-2 focus:ring-purple-400/50"
                disabled={isBusy}
              >
                <option value="A">A</option>
                <option value="B">B</option>
                <option value="C">C</option>
                <option value="D">D</option>
              </select>
            </label>
            <button
              type="button"
              onClick={() => removeQuestion(i)}
              disabled={questions.length === 1 || isBusy}
              className="mt-2 bg-red-600 text-white border-none rounded px-3 py-1 cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
              aria-label={`Eliminar pregunta ${i + 1}`}
            >
              Eliminar pregunta
            </button>
          </div>
        ))}
        <button
          type="button"
          onClick={addQuestion}
          className="bg-purple-600 text-white border-none rounded px-4 py-2 cursor-pointer mt-4"
          disabled={isBusy}
        >
          Añadir pregunta
        </button>
      </fieldset>

      <button
        type="submit"
        disabled={isBusy}
        className="w-full py-3 text-lg font-bold mt-8 rounded-md bg-purple-600 hover:bg-purple-700 transition-colors disabled:opacity-70 disabled:cursor-not-allowed"
      >
        {loading
          ? examToEdit
            ? 'Guardando...'
            : 'Creando...'
          : examToEdit
          ? 'Guardar cambios'
          : 'Crear examen'}
      </button>
    </form>
  );
}
