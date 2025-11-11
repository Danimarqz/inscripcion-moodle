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

const DEFAULT_QUESTION: QuestionCreate = { correct_option: 'A', is_active: true, is_cancelled: false };
const VALID_OPTIONS = ['A', 'B', 'C', 'D'];

export default function ExamForm({ examId }: ExamFormProps) {
  const { token, loading: authenticating, error: authError } = useAdminAuth();
  const { loading, error, run, setError } = useAsyncTask();
  const { questions, setAll, updateQuestion, addQuestion, removeQuestion } =
    useQuestionList<QuestionCreate | QuestionEdit>([{ ...DEFAULT_QUESTION }]);

  const [examToEdit, setExamToEdit] = useState<ExamEdit | null>(null);
  const [name, setName] = useState('');
  const [isActive, setIsActive] = useState(false);
  const [showScore, setShowScore] = useState(false);
  const [showPercentile, setShowPercentile] = useState(false);
  const [showScoreFull, setShowScoreFull] = useState(false);
  const [validatedTribunal, setValidatedTribunal] = useState(false);

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
      setShowScore(false);
      setShowPercentile(false);
      setShowScoreFull(false);
      setValidatedTribunal(false);
      setAll([{ ...DEFAULT_QUESTION }]);
      setError(null);
      return;
    }

    void run(async () => {
      const examData = await getExamById(examId, token);
      setExamToEdit(examData);
      setName(examData.name ?? '');
      setIsActive(Boolean(examData.is_active));
      setShowScore(Boolean(examData.show_score));
      setShowPercentile(Boolean(examData.show_percentile));
      setShowScoreFull(Boolean(examData.show_score_full));
      setValidatedTribunal(Boolean(examData.validated_tribunal));

      const normalizedQuestions = (examData.questions.length
        ? examData.questions
        : [{ ...DEFAULT_QUESTION } as QuestionEdit]
      ).map((question) => ({
        ...question,
        is_active: question.is_active ?? true,
        is_cancelled: question.is_cancelled ?? false,
      }));

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

    const activeCount = questions.filter(
      (q) => q.is_active !== false && q.is_cancelled !== true,
    ).length;
    if (activeCount === 0) {
      setError('Debe haber al menos una pregunta activa no anulada.');
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
      show_score: showScore,
      show_percentile: showPercentile,
      show_score_full: showScoreFull,
      validated_tribunal: validatedTribunal,
      questions: questions.map((q) => ({
        id: 'id' in q ? q.id : undefined,
        correct_option: q.correct_option.toUpperCase(),
        is_active: q.is_active !== false,
        is_cancelled: q.is_cancelled === true,
        name: typeof q.name === 'number' ? q.name : undefined,
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
    const isCancelled = question.is_cancelled === true;
    const displayName = question.name ?? position;
    const label = `${isReserve ? 'Reserva' : 'Pregunta'} ${displayName}`;
    const badgeConfig = isCancelled
      ? {
          text: 'Anulada',
          className: 'bg-red-500/20 text-red-300 border border-red-500/50',
        }
      : {
          text: isReserve ? 'Reserva' : 'Activa',
          className: isReserve
            ? 'bg-amber-500/20 text-amber-300 border border-amber-500/50'
            : 'bg-green-500/20 text-green-300 border border-green-500/50',
        };

    return (
      <div
        key={key}
        className="bg-[#2a2d33] p-6 mb-4 rounded-lg shadow-lg border border-transparent hover:border-brand-blue/40 transition-colors"
      >
        <div className="flex items-center justify-between mb-3">
          <span className="font-bold text-lg text-brand-pink">{label}</span>
          <span className={`text-xs font-semibold px-2 py-1 rounded ${badgeConfig.className}`}>
            {badgeConfig.text}
          </span>
        </div>
        {'id' in question && question.id ? (
          <p className="text-xs text-gray-400 mb-2">ID interno: {question.id}</p>
        ) : null}
        {isCancelled && (
          <p className="text-sm text-red-300 mb-4">
            Esta pregunta se ha marcado como anulada y no contará para la nota.
          </p>
        )}
        <label className="block font-bold text-brand-pink mb-2">
          Opcion correcta:
          <select
            value={(question.correct_option ?? 'A').toUpperCase()}
            onChange={(e) => updateQuestion(index, 'correct_option', e.currentTarget.value)}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
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
            className="bg-brand-blue text-white border-none rounded px-3 py-1 cursor-pointer hover:bg-brand-blue/80 disabled:opacity-60 disabled:cursor-not-allowed"
            disabled={isBusy}
          >
            {isReserve ? 'Activar' : 'Mover a reserva'}
          </button>
          <button
            type="button"
            onClick={() => updateQuestion(index, 'is_cancelled', !isCancelled)}
            className="bg-yellow-500 text-[#1a1c22] border-none rounded px-3 py-1 cursor-pointer font-semibold hover:bg-yellow-400 disabled:opacity-60 disabled:cursor-not-allowed"
            disabled={isBusy}
          >
            {isCancelled ? 'Quitar anulación' : 'Marcar como anulada'}
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
        <label className="block font-bold text-brand-pink mb-2">
          Nombre del examen:
          <input
            type="text"
            value={name}
            onInput={(e) => setName((e.target as HTMLInputElement).value)}
            required
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
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
        <label className="font-bold text-brand-pink">Activo</label>
      </div>

      <fieldset className="mb-6 border border-[#444] rounded-lg p-4" disabled={isBusy}>
        <legend className="font-bold text-brand-pink px-2">Resultados visibles para el alumno</legend>
        <p className="text-xs text-gray-400 mb-4">
          Activa cada opción de forma independiente. Si ninguna está marcada, el alumno solo verá un mensaje de confirmación.
        </p>
        <div className="flex flex-col gap-3">
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={showScore}
              onChange={(e) => setShowScore(e.currentTarget.checked)}
              className="mr-1"
              disabled={isBusy}
            />
            <span className="font-bold text-brand-pink">Mostrar nota final (porcentaje)</span>
          </label>
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={showPercentile}
              onChange={(e) => setShowPercentile(e.currentTarget.checked)}
              className="mr-1"
              disabled={isBusy}
            />
            <span className="font-bold text-brand-pink">
              Mostrar percentil y posición entre los exámenes entregados
            </span>
          </label>
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={showScoreFull}
              onChange={(e) => setShowScoreFull(e.currentTarget.checked)}
              className="mr-1"
              disabled={isBusy}
            />
            <span className="font-bold text-brand-pink">Mostrar detalle de aciertos (ej. "20 de 80 preguntas")</span>
          </label>
          <div className="flex flex-col gap-1">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={validatedTribunal}
                onChange={(e) => setValidatedTribunal(e.currentTarget.checked)}
                className="mr-1"
                disabled={isBusy}
              />
              <span className="font-bold text-brand-pink">
                Validado por tribunal (muestra respuestas correctas en la consulta r&aacute;pida)
              </span>
            </label>
            <p className="text-xs text-gray-400 ml-6">
              Al activarlo, los alumnos ver&aacute;n qu&eacute; marcaron frente a la respuesta oficial cuando consulten su nota.
            </p>
          </div>
        </div>
      </fieldset>

      <fieldset className="border border-[#444] p-4 rounded-lg" disabled={isBusy}>
        <legend className="font-bold text-xl mb-4 text-brand-pink">Preguntas</legend>

        <section>
          <h3 className="text-lg font-semibold text-brand-pink mb-3">Preguntas activas ({activeEntries.length})</h3>
          {activeEntries.length === 0 ? (
            <p className="text-sm text-red-400 bg-red-500/10 border border-red-500/40 p-3 rounded">
              Debe haber al menos una pregunta activa para poder publicar el examen.
            </p>
          ) : (
            activeEntries.map((entry, idx) => renderQuestionCard(entry, idx + 1, false))
          )}
        </section>

        <section className="mt-6">
          <h3 className="text-lg font-semibold text-brand-pink mb-2">Preguntas de reserva ({reserveEntries.length})</h3>
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
          className="bg-brand-blue text-white border-none rounded px-4 py-2 cursor-pointer mt-6 hover:bg-brand-blue/80"
          disabled={isBusy}
        >
          Anadir pregunta
        </button>
      </fieldset>

      <button
        type="submit"
        disabled={isBusy}
        className="w-full py-3 text-lg font-bold mt-8 cursor-pointer rounded-md bg-brand-blue hover:bg-brand-blue/80 transition-colors disabled:opacity-70 disabled:cursor-not-allowed"
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
