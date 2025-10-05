import { useEffect, useMemo, useState } from 'preact/hooks';
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

const DEFAULT_QUESTION: QuestionCreate = { correct_option: 'A', is_active: true };
const VALID_OPTIONS = ['A', 'B', 'C', 'D'];

export default function ExamForm({ examId }: ExamFormProps) {
  const { token, loading: authenticating, error: authError } = useAdminAuth();
  const { loading, error, run, setError } = useAsyncTask();
  const { questions, setAll, updateQuestion, addQuestion, removeQuestion } =
    useQuestionList<QuestionCreate | QuestionEdit>([{ ...DEFAULT_QUESTION }]);

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
      setAll([{ ...DEFAULT_QUESTION }]);
      setError(null);
      return;
    }

    void run(async () => {
      const examData = await getExamById(examId, token);
      setExamToEdit(examData);
      setName(examData.name ?? '');
      setIsActive(Boolean(examData.is_active));
      setShowResponse(Boolean(examData.show_response));

      const normalizedQuestions = (examData.questions.length
        ? examData.questions
        : [{ ...DEFAULT_QUESTION } as QuestionEdit]
      ).map((question) => ({ ...question, is_active: question.is_active ?? true }));

      setAll(normalizedQuestions as (QuestionCreate | QuestionEdit)[]);
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

    const activeCount = questions.filter((q) => q.is_active !== false).length;
    if (activeCount === 0) {
      setError('Debe haber al menos una pregunta activa.');
      return;
    }

    if (questions.some((q) => !VALID_OPTIONS.includes(q.correct_option.toUpperCase()))) {
      setError('Todas las preguntas deben tener una opcion correcta valida (A, B, C o D).');
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
        is_active: q.is_active !== false,
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
      // Error ya gestionado por run(); no hay nada mas que hacer aqui.
    }
  }

  const isBusy = loading || authenticating;
  const formError = authError || error;

  const questionEntries = useMemo(
    () => questions.map((question, index) => ({ index, question })),
    [questions],
  );

  const activeEntries = questionEntries.filter(({ question }) => question.is_active !== false);
  const reserveEntries = questionEntries.filter(({ question }) => question.is_active === false);

  function handleToggleState(index: number, nextState: boolean) {
    updateQuestion(index, 'is_active', nextState);
  }

  function renderQuestionCard(
    entry: { index: number; question: QuestionCreate | QuestionEdit },
    position: number,
    isReserve: boolean,
  ) {
    const { index, question } = entry;
    const key = question.id ?? `${isReserve ? 'reserve' : 'active'}-${index}`;
    const badgeClass = isReserve
      ? 'bg-amber-500/20 text-amber-300 border border-amber-500/50'
      : 'bg-green-500/20 text-green-300 border border-green-500/50';
    const label = isReserve ? `Reserva ${position}` : `Pregunta ${position}`;

    return (
      <div
        key={key}
        className="bg-[#2a2d33] p-6 mb-4 rounded-lg shadow-lg border border-transparent hover:border-purple-500/40 transition-colors"
      >
        <div className="flex items-center justify-between mb-3">
          <span className="font-bold text-lg text-purple-200">{label}</span>
          <span className={`text-xs font-semibold px-2 py-1 rounded ${badgeClass}`}>
            {isReserve ? 'Reserva' : 'Activa'}
          </span>
        </div>
        {'id' in question && question.id ? (
          <p className="text-xs text-gray-400 mb-2">ID interno: {question.id}</p>
        ) : null}
        <label className="block font-bold text-purple-500 mb-2">
          Opcion correcta:
          <select
            value={(question.correct_option ?? 'A').toUpperCase()}
            onChange={(e) => updateQuestion(index, 'correct_option', e.currentTarget.value)}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-purple-400 focus:ring-2 focus:ring-purple-400/50"
            disabled={isBusy}
          >
            {VALID_OPTIONS.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        </label>
        <div className="flex flex-wrap gap-3 mt-4">
          <button
            type="button"
            onClick={() => removeQuestion(index)}
            disabled={questions.length === 1 || isBusy}
            className="bg-red-600 text-white border-none rounded px-3 py-1 cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
            aria-label={`Eliminar ${label.toLowerCase()}`}
          >
            Eliminar
          </button>
          <button
            type="button"
            onClick={() => handleToggleState(index, isReserve)}
            className="bg-purple-600 text-white border-none rounded px-3 py-1 cursor-pointer hover:bg-purple-700 disabled:opacity-60 disabled:cursor-not-allowed"
            disabled={isBusy}
          >
            {isReserve ? 'Activar' : 'Mover a reserva'}
          </button>
        </div>
      </div>
    );
  }

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
        <legend className="font-bold text-xl mb-4 text-purple-200">Preguntas</legend>

        <section>
          <h3 className="text-lg font-semibold text-purple-300 mb-3">Preguntas activas ({activeEntries.length})</h3>
          {activeEntries.length === 0 ? (
            <p className="text-sm text-red-400 bg-red-500/10 border border-red-500/40 p-3 rounded">
              Debe haber al menos una pregunta activa para poder publicar el examen.
            </p>
          ) : (
            activeEntries.map((entry, idx) => renderQuestionCard(entry, idx + 1, false))
          )}
        </section>

        <section className="mt-6">
          <h3 className="text-lg font-semibold text-purple-300 mb-2">Preguntas de reserva ({reserveEntries.length})</h3>
          <p className="text-xs text-gray-400 mb-4">
            Las preguntas de reserva no puntuan salvo que sustituyan a una activa invalidada.
          </p>
          {reserveEntries.length === 0 ? (
            <p className="text-sm text-gray-400 bg-gray-500/10 border border-gray-500/30 p-3 rounded">
              No hay preguntas de reserva configuradas.
            </p>
          ) : (
            reserveEntries.map((entry, idx) => renderQuestionCard(entry, idx + 1, true))
          )}
        </section>

        <button
          type="button"
          onClick={addQuestion}
          className="bg-purple-600 text-white border-none rounded px-4 py-2 cursor-pointer mt-6 hover:bg-purple-700"
          disabled={isBusy}
        >
          Anadir pregunta
        </button>
      </fieldset>

      <button
        type="submit"
        disabled={isBusy}
        className="w-full py-3 text-lg font-bold mt-8 cursor-pointer rounded-md bg-purple-600 hover:bg-purple-700 transition-colors disabled:opacity-70 disabled:cursor-not-allowed"
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
