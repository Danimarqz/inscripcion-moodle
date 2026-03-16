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
import QuestionCardFrame from './questions/QuestionCardFrame';
import QuestionSection from './questions/QuestionSection';
import { ANSWER_OPTIONS, type AnswerOption } from '../constants/answerOptions';

interface ExamFormProps {
  examId?: number;
}

const DEFAULT_QUESTION: QuestionCreate = { correct_option: 'A', is_active: true, is_cancelled: false };

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
  const [subtractsPoints, setSubtractsPoints] = useState(false);
  const [penaltyValue, setPenaltyValue] = useState(0);
  const [maxScore, setMaxScore] = useState(100);
  const [secondaryMaxScores, setSecondaryMaxScores] = useState('');
  const [passingCriteriaType, setPassingCriteriaType] = useState('disabled');
  const [passingCriteriaValue, setPassingCriteriaValue] = useState<number | null>(null);

  useEffect(() => {
    if (authenticating) return;

    if (
      questions.some((q) => {
        const normalized = ((q.correct_option ?? 'A').toUpperCase() as AnswerOption);
        return !ANSWER_OPTIONS.includes(normalized);
      })
    ) {
      setError('Todas las preguntas deben tener una opcion correcta valida (A, B, C o D).');
      return;
    }

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
      setSubtractsPoints(false);
      setPenaltyValue(0);
      setMaxScore(100);
      setSecondaryMaxScores('');
      setPassingCriteriaType('disabled');
      setPassingCriteriaValue(null);
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
      setSubtractsPoints(Boolean(examData.subtracts_points));
      setPenaltyValue(examData.penalty_value ?? 0);
      setMaxScore(examData.max_score ?? 100);
      setSecondaryMaxScores(examData.secondary_max_scores ?? '');
      setPassingCriteriaType(examData.passing_criteria_type ?? 'disabled');
      setPassingCriteriaValue(examData.passing_criteria_value ?? null);

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

    if (
      questions.some((q) => {
        const normalized = ((q.correct_option ?? 'A').toUpperCase() as AnswerOption);
        return !ANSWER_OPTIONS.includes(normalized);
      })
    ) {
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
      subtracts_points: subtractsPoints,
      penalty_value: subtractsPoints ? penaltyValue : 0,
      max_score: maxScore,
      secondary_max_scores: secondaryMaxScores,
      passing_criteria_type: passingCriteriaType,
      passing_criteria_value: passingCriteriaType !== 'disabled' ? passingCriteriaValue : null,
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

    const meta = (
      <>
        {'id' in question && question.id ? (
          <p className="text-xs text-gray-400 mb-2">ID interno: {question.id}</p>
        ) : null}
        {isCancelled && (
          <p className="text-sm text-red-300 mb-4">
            Esta pregunta se ha marcado como anulada y no contará para la nota.
          </p>
        )}
      </>
    );

    return (
      <QuestionCardFrame
        key={key}
        label={label}
        badgeText={badgeConfig.text}
        badgeClassName={badgeConfig.className}
        meta={meta}
      >
        <label className="block font-bold text-brand-pink mb-2">
          Opcion correcta:
          <select
            value={(question.correct_option ?? 'A').toUpperCase()}
            onChange={(e) => updateQuestion(index, 'correct_option', e.currentTarget.value)}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
            disabled={isBusy}
          >
            {ANSWER_OPTIONS.map((option) => (
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
      </QuestionCardFrame>
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

      <fieldset className="mb-6 border border-[#444] rounded-lg p-4" disabled={isBusy}>
        <legend className="font-bold text-brand-pink px-2">Configuración de corrección</legend>
        <div className="flex flex-col gap-4">
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={subtractsPoints}
              onChange={(e) => {
                const checked = e.currentTarget.checked;
                setSubtractsPoints(checked);
                if (!checked) setPenaltyValue(0);
                else if (penaltyValue === 0) setPenaltyValue(0.25);
              }}
              className="mr-1"
              disabled={isBusy}
            />
            <span className="font-bold text-brand-pink">Las preguntas mal contestadas restan puntos</span>
          </label>
          
          {subtractsPoints && (
             <label className="flex items-center gap-2 ml-6">
               <span className="text-white">Cantidad a restar por fallo:</span>
               <select
                 value={penaltyValue}
                 onChange={(e) => setPenaltyValue(parseFloat(e.currentTarget.value))}
                 className="px-3 py-1 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
                 disabled={isBusy}
               >
                 <option value={0}>0</option>
                 <option value={0.25}>0.25</option>
                 <option value={0.33333333}>0.33</option>
                 <option value={0.5}>0.5</option>
               </select>
             </label>
          )}
        </div>
      </fieldset>


      <fieldset className="mb-6 border border-[#444] rounded-lg p-4" disabled={isBusy}>
        <legend className="font-bold text-brand-pink px-2">Configuración de puntuación</legend>
        <div className="flex flex-col gap-4">
          <label className="block text-brand-pink font-bold">
            Puntuación máxima (Base):
            <input
              type="number"
              value={maxScore}
              onInput={(e) => setMaxScore(parseFloat((e.target as HTMLInputElement).value))}
              className="w-full mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
              min="1"
            />
            <span className="text-xs text-gray-400 block mt-1">
              Por defecto es 100. Cambia esto si quieres que el examen sea sobre 10, 60, etc.
            </span>
          </label>

          <label className="block text-brand-pink font-bold">
            Otras bases para mostrar (opcional):
            <input
              type="text"
              value={secondaryMaxScores}
              onInput={(e) => setSecondaryMaxScores((e.target as HTMLInputElement).value)}
              className="w-full mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
              placeholder="Ej: 10,60"
            />
            <span className="text-xs text-gray-400 block mt-1">
              Separa con comas. Ej: "10,60" mostrará también la nota sobre 10 y sobre 60.
            </span>
          </label>
        </div>
      </fieldset>

      <fieldset className="mb-6 border border-[#444] rounded-lg p-4" disabled={isBusy}>
        <legend className="font-bold text-brand-pink px-2">Criterio para aprobar</legend>
        <p className="text-xs text-gray-400 mb-4">
          Configura si quieres informar a los alumnos de si han aprobado y permitirles añadir méritos.
        </p>
        <div className="flex flex-col gap-4">
          <label className="block text-brand-pink font-bold">
            Tipo de criterio:
            <select
              value={passingCriteriaType}
              onChange={(e) => {
                const val = (e.target as HTMLSelectElement).value;
                setPassingCriteriaType(val);
                if (val === 'disabled') setPassingCriteriaValue(null);
              }}
              className="w-full mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
              disabled={isBusy}
            >
              <option value="disabled">Desactivado</option>
              <option value="min_score">Nota minima</option>
              <option value="top10_pct">% de la media del top 10</option>
            </select>
          </label>
          {passingCriteriaType !== 'disabled' && (
            <label className="block text-brand-pink font-bold">
              {passingCriteriaType === 'min_score' ? 'Nota minima para aprobar:' : 'Porcentaje de la media del top 10:'}
              <input
                type="number"
                step="0.01"
                min="0"
                value={passingCriteriaValue ?? ''}
                onInput={(e) => {
                  const v = (e.target as HTMLInputElement).value;
                  setPassingCriteriaValue(v ? parseFloat(v) : null);
                }}
                className="w-full mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
                disabled={isBusy}
              />
              <span className="text-xs text-gray-400 block mt-1">
                {passingCriteriaType === 'min_score'
                  ? 'Los alumnos con nota >= este valor se consideran aprobados.'
                  : 'Ej: 80 significa que el umbral es el 80% de la media del top 10.'}
              </span>
            </label>
          )}
        </div>
      </fieldset>

      <fieldset className="border border-[#444] p-4 rounded-lg" disabled={isBusy}>
        <legend className="font-bold text-xl mb-4 text-brand-pink">Preguntas</legend>

        <QuestionSection
          title="Preguntas activas"
          entries={activeEntries}
          emptyMessage="Debe haber al menos una pregunta activa para poder publicar el examen."
          renderEntry={(entry, position) => renderQuestionCard(entry, position, false)}
        />

        <QuestionSection
          title="Preguntas de reserva"
          description="Las preguntas de reserva no puntuan salvo que sustituyan a una activa invalidada."
          entries={reserveEntries}
          emptyMessage="No hay preguntas de reserva configuradas."
          emptyMessageClassName="text-sm text-gray-400 bg-gray-500/10 border border-gray-500/30 p-3 rounded"
          renderEntry={(entry, position) => renderQuestionCard(entry, position, true)}
        />

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
