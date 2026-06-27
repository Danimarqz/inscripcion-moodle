import { useEffect, useMemo, useRef, useState } from 'preact/hooks';
import type {
  Exam,
  ExamCreateWithQuestions,
  ExamEdit,
  QuestionCreate,
  QuestionEdit,
  QuestionGroup,
} from '../types/exam';
import { createExam, editExam, getAdminExams, getExamById, getExamQuestions, getQuestionFeedbackVideo, patchQuestionFeedbackVideo } from '../services/adminService';
import { useAdminAuth } from '../hooks/useAdminAuth';
import { useQuestionList } from '../hooks/useQuestionList';
import { useAsyncTask } from '../hooks/useAsyncTask';
import QuestionCardFrame from './questions/QuestionCardFrame';
import AssociateExamsModal from './modals/AssociateExamsModal';
import { ANSWER_OPTIONS, type AnswerOption } from '../constants/answerOptions';

interface ExamFormProps {
  examId?: number;
}

const DEFAULT_QUESTION: QuestionCreate = { correct_option: 'A', is_active: true, is_cancelled: false, name: 1 };

export default function ExamForm({ examId }: ExamFormProps) {
  const { token, loading: authenticating, error: authError } = useAdminAuth();
  const { loading, error, run, setError } = useAsyncTask();
  const { questions, setAll, updateQuestion, addQuestion } =
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
  const [scoringMode, setScoringMode] = useState<'legacy' | 'absolute'>('legacy');
  const [pointsPerCorrect, setPointsPerCorrect] = useState(0.4);
  const [pointsPerWrong, setPointsPerWrong] = useState(0.1);
  const [maxScore, setMaxScore] = useState(100);
  const [secondaryMaxScores, setSecondaryMaxScores] = useState('');
  const [passingCriteriaType, setPassingCriteriaType] = useState('disabled');
  const [passingCriteriaValue, setPassingCriteriaValue] = useState<number | null>(null);
  const [examWeight, setExamWeight] = useState(0.5);
  const [maxMerits, setMaxMerits] = useState(100);
  const [skipWeights, setSkipWeights] = useState(false);
  const [raffleEnabled, setRaffleEnabled] = useState(false);
  const [raffleTerms, setRaffleTerms] = useState('');
  const [associatedExamIds, setAssociatedExamIds] = useState<number[]>([]);
  const [otherExams, setOtherExams] = useState<Exam[]>([]);
  const [associateModalOpen, setAssociateModalOpen] = useState(false);
  const [displayWeightOverride, setDisplayWeightOverride] = useState(false);
  const [displayExamWeight, setDisplayExamWeight] = useState(0.5);
  // Question groups (e.g. Teórico / Práctico). Empty = flat exam (legacy behaviour).
  // group.position is a STABLE id (never reused) so removing a group never renumbers
  // the others, keeping each question's group_position link valid.
  const [groups, setGroups] = useState<QuestionGroup[]>([]);
  const grouped = groups.length > 0;

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
      setScoringMode('legacy');
      setPointsPerCorrect(0.4);
      setPointsPerWrong(0.1);
      setMaxScore(100);
      setSecondaryMaxScores('');
      setPassingCriteriaType('disabled');
      setPassingCriteriaValue(null);
      setExamWeight(0.5);
      setMaxMerits(100);
      setGroups([]);
      setAll([{ ...DEFAULT_QUESTION }]);
      setError(null);
      return;
    }

    void run(async () => {
      const [examData, examQuestions] = await Promise.all([
        getExamById(examId, token),
        getExamQuestions(examId, token),
      ]);
      setExamToEdit(examData);
      setName(examData.name ?? '');
      setIsActive(Boolean(examData.is_active));
      setShowScore(Boolean(examData.show_score));
      setShowPercentile(Boolean(examData.show_percentile));
      setShowScoreFull(Boolean(examData.show_score_full));
      setValidatedTribunal(Boolean(examData.validated_tribunal));
      setSubtractsPoints(Boolean(examData.subtracts_points));
      setPenaltyValue(examData.penalty_value ?? 0);
      setScoringMode((examData.scoring_mode as 'legacy' | 'absolute') ?? 'legacy');
      setPointsPerCorrect(examData.points_per_correct ?? 0.4);
      setPointsPerWrong(examData.points_per_wrong ?? 0.1);
      setMaxScore(examData.max_score ?? 100);
      setSecondaryMaxScores(examData.secondary_max_scores ?? '');
      setPassingCriteriaType(examData.passing_criteria_type ?? 'disabled');
      setPassingCriteriaValue(examData.passing_criteria_value ?? null);
      setExamWeight(examData.exam_weight ?? 0.5);
      setMaxMerits(examData.max_merits ?? 100);
      setSkipWeights(Boolean(examData.skip_weights));
      setRaffleEnabled(Boolean(examData.raffle_enabled));
      setRaffleTerms(examData.raffle_terms ?? '');
      setAssociatedExamIds(examData.associated_exam_ids ?? []);
      setDisplayWeightOverride(examData.display_exam_weight != null);
      setDisplayExamWeight(examData.display_exam_weight ?? examData.exam_weight ?? 0.5);

      const loadedGroups = examData.groups ?? [];
      setGroups(loadedGroups);
      // Backend returns each question's group_id; map it to the group's stable
      // position so the per-question group selector can bind to group_position.
      const idToPosition = new Map<number, number>();
      for (const g of loadedGroups) {
        if (g.id != null) idToPosition.set(g.id, g.position);
      }

      const normalizedQuestions = (examQuestions.length
        ? examQuestions
        : [{ ...DEFAULT_QUESTION } as QuestionEdit]
      ).map((question, idx) => {
        const groupId = (question as { group_id?: number | null }).group_id;
        return {
          ...question,
          is_active: question.is_active ?? true,
          is_cancelled: question.is_cancelled ?? false,
          name: typeof question.name === 'number' && question.name > 0 ? question.name : idx + 1,
          group_position: groupId != null ? idToPosition.get(groupId) ?? null : null,
        };
      });

      setAll(normalizedQuestions as (QuestionCreate | QuestionEdit)[]);
    }).catch(() => undefined);
  }, [authenticating, examId, run, setAll, setError, token]);

  // Load other exams to offer as percentile-group candidates (edit only).
  useEffect(() => {
    if (authenticating || !token || !examId) return;
    getAdminExams(token)
      .then((exams) => setOtherExams(exams.filter((e) => e.id !== examId)))
      .catch(() => undefined);
  }, [authenticating, examId, token]);

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

    const names = questions.map((q) => q.name);
    if (names.some((n) => typeof n !== 'number' || !Number.isInteger(n) || (n as number) <= 0)) {
      setError('Cada pregunta debe tener un número entero positivo en el campo Nº.');
      return;
    }
    const sortedNames = [...(names as number[])].sort((a, b) => a - b);
    for (let i = 0; i < sortedNames.length; i += 1) {
      if (sortedNames[i] !== i + 1) {
        setError(
          `Los números de pregunta deben formar la secuencia 1..${questions.length} sin huecos ni duplicados.`,
        );
        return;
      }
    }

    if (grouped) {
      for (const g of groups) {
        if (!g.name.trim() || !(g.max_score > 0) || g.points_per_wrong < 0) {
          setError('Cada grupo necesita un nombre, una valoración mayor que 0 y una penalización no negativa.');
          return;
        }
        if (g.min_passing_score != null && (g.min_passing_score < 0 || g.min_passing_score > g.max_score)) {
          setError(`La nota mínima del grupo "${g.name || '?'}" debe estar entre 0 y su valoración.`);
          return;
        }
      }
      const validPositions = new Set(groups.map((g) => g.position));
      const unassigned = questions.some(
        (q) =>
          q.is_active !== false &&
          q.is_cancelled !== true &&
          (q.group_position == null || !validPositions.has(q.group_position)),
      );
      if (unassigned) {
        setError('Con grupos activos, cada pregunta activa no anulada debe asignarse a un grupo.');
        return;
      }
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
      scoring_mode: scoringMode,
      ...(scoringMode === 'absolute'
        ? { points_per_correct: pointsPerCorrect, points_per_wrong: pointsPerWrong }
        : {}),
      max_score: maxScore,
      secondary_max_scores: secondaryMaxScores,
      passing_criteria_type: passingCriteriaType,
      passing_criteria_value: passingCriteriaType !== 'disabled' ? passingCriteriaValue : null,
      exam_weight: examWeight,
      max_merits: maxMerits,
      skip_weights: skipWeights,
      raffle_enabled: raffleEnabled,
      raffle_terms: raffleEnabled ? raffleTerms : '',
      ...(examToEdit ? { associated_exam_ids: associatedExamIds } : {}),
      display_exam_weight: displayWeightOverride ? displayExamWeight : null,
      ...(examToEdit && !displayWeightOverride ? { clear_display_weight: true } : {}),
      groups: grouped
        ? groups.map((g) => ({
            id: g.id,
            name: g.name.trim(),
            position: g.position,
            max_score: g.max_score,
            points_per_wrong: g.points_per_wrong,
            min_passing_score: g.min_passing_score ?? null,
            eliminatory: g.eliminatory,
          }))
        : [],
      questions: questions.map((q) => ({
        id: 'id' in q ? q.id : undefined,
        correct_option: q.correct_option.toUpperCase(),
        is_active: q.is_active !== false,
        is_cancelled: q.is_cancelled === true,
        name: q.name as number,
        label: q.label ?? null,
        group_position: grouped ? q.group_position ?? null : null,
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
  const errorRef = useRef<HTMLParagraphElement | null>(null);

  // Scroll the error banner into view whenever a new error appears, so an
  // admin who clicked Guardar from the bottom of a long form actually sees
  // why it failed instead of silently staring at an unchanged page.
  useEffect(() => {
    if (formError && errorRef.current) {
      errorRef.current.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }, [formError]);

  // Render in array insertion order, NOT sorted by `name`. Sorting while the
  // admin is typing into a Nº input would make the card jump out of view and
  // steal focus. Backend returns questions already sorted by name on load, so
  // the initial display is correct; the admin edits numbers in place and the
  // student view (which is read-only) is what actually renders in numeric
  // order.
  const questionEntries = useMemo(
    () => questions.map((question, index) => ({ index, question })),
    [questions],
  );

  function handleToggleState(index: number, nextState: boolean) {
    updateQuestion(index, 'is_active', nextState);
  }

  function handleAddQuestion() {
    const nextNumber = questions.length + 1;
    addQuestion({ name: nextNumber });
  }

  // Remove a question and renumber the survivors so the set stays 1..N-1.
  function handleRemoveQuestion(index: number) {
    const removed = questions[index];
    const removedName = typeof removed?.name === 'number' ? removed.name : null;
    const remaining = questions
      .filter((_, i) => i !== index)
      .map((q) => {
        if (removedName == null || typeof q.name !== 'number' || q.name <= removedName) {
          return q;
        }
        return { ...q, name: q.name - 1 };
      });
    setAll(remaining);
  }

  function nextGroupPosition() {
    return groups.length ? Math.max(...groups.map((g) => g.position)) + 1 : 0;
  }

  function handleAddGroup() {
    setGroups([
      ...groups,
      {
        name: '',
        position: nextGroupPosition(),
        max_score: 0,
        points_per_wrong: 0.25,
        min_passing_score: null,
        eliminatory: true,
      },
    ]);
  }

  function updateGroup(index: number, patch: Partial<QuestionGroup>) {
    setGroups(groups.map((g, i) => (i === index ? { ...g, ...patch } : g)));
  }

  // Remove a group and detach any question that pointed at it (positions are
  // stable, so the surviving groups' links stay valid).
  function handleRemoveGroup(index: number) {
    const removedPos = groups[index].position;
    setGroups(groups.filter((_, i) => i !== index));
    setAll(
      questions.map((q) =>
        q.group_position === removedPos ? { ...q, group_position: null } : q,
      ),
    );
  }

  function renderQuestionCard(entry: { index: number; question: QuestionCreate | QuestionEdit }) {
    const { index, question } = entry;
    const isReserve = question.is_active === false;
    const key = question.id ?? `q-${index}`;
    const isCancelled = question.is_cancelled === true;
    const customLabel = typeof question.label === 'string' ? question.label.trim() : '';
    const label = customLabel
      ? customLabel
      : `${isReserve ? 'Reserva' : 'Pregunta'} ${question.name ?? '?'}`;
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
        <div className="flex flex-wrap gap-4 mb-3">
          <label className="block font-bold text-brand-pink">
            Nº:
            <input
              type="number"
              min={1}
              step={1}
              value={typeof question.name === 'number' ? question.name : ''}
              onInput={(e) => {
                const raw = (e.target as HTMLInputElement).value;
                const parsed = parseInt(raw, 10);
                updateQuestion(index, 'name', Number.isFinite(parsed) && parsed > 0 ? parsed : undefined);
              }}
              className="block w-24 mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
              disabled={isBusy}
            />
          </label>
          <label className="block font-bold text-brand-pink">
            Etiqueta:
            <input
              type="text"
              maxLength={32}
              placeholder="R1, Extra…"
              value={question.label ?? ''}
              onInput={(e) => {
                const raw = (e.currentTarget as HTMLInputElement).value;
                const trimmed = raw.trim();
                updateQuestion(index, 'label', trimmed === '' ? null : raw);
              }}
              className="block w-40 mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
              disabled={isBusy}
            />
          </label>
        </div>
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
        {grouped && (
          <label className="block font-bold text-brand-pink mb-2">
            Grupo:
            <select
              value={question.group_position ?? ''}
              onChange={(e) => {
                const raw = e.currentTarget.value;
                updateQuestion(index, 'group_position', raw === '' ? null : parseInt(raw, 10));
              }}
              className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
              disabled={isBusy}
            >
              <option value="">— Sin asignar —</option>
              {groups.map((g, gi) => (
                <option key={g.position} value={g.position}>
                  {g.name.trim() || `Grupo ${gi + 1}`}
                </option>
              ))}
            </select>
          </label>
        )}
        <div className="flex flex-wrap gap-3 mt-4">
          <button
            type="button"
            onClick={() => handleRemoveQuestion(index)}
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
        {'id' in question && question.id && token && (
          <FeedbackVideoKeyField questionId={question.id} token={token} />
        )}
      </QuestionCardFrame>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="max-w-3xl mx-auto text-white font-sans">
      <h2 className="text-3xl font-bold text-center mb-8">{examToEdit ? 'Editar Examen' : 'Crear Examen'}</h2>

      {formError && (
        <p
          ref={errorRef}
          role="alert"
          aria-live="assertive"
          tabIndex={-1}
          className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mb-6"
        >
          {formError}
        </p>
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

      <div className="mb-6 flex items-center flex-wrap gap-3">
        <label className="flex items-center cursor-pointer">
          <input
            type="checkbox"
            checked={isActive}
            onChange={(e) => setIsActive(e.currentTarget.checked)}
            className="mr-2"
            disabled={isBusy}
          />
          <span className="font-bold text-brand-pink">Activo</span>
        </label>
        {examToEdit && (
          <>
            <span className="ml-auto text-sm text-gray-400">
              {associatedExamIds.length > 0
                ? `${associatedExamIds.length} examen(es) asociado(s)`
                : 'Ninguno asociado'}
            </span>
            <button
              type="button"
              onClick={() => setAssociateModalOpen(true)}
              disabled={isBusy}
              className="px-4 py-2 rounded font-semibold bg-brand-blue text-white hover:bg-brand-blue/80 transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
            >
              Asociar exámenes para percentil
            </button>
          </>
        )}
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
        <legend className="font-bold text-brand-pink px-2">Grupos de preguntas</legend>
        <p className="text-xs text-gray-400 mb-4">
          Opcional. Crea grupos (ej. Teórico, Práctico) para puntuar por bloques. Cada grupo
          se corrige en modo absoluto: puntos por acierto = valoración ÷ nº de preguntas activas
          del grupo, menos la penalización por fallo. La nota total es la suma de los grupos y, si
          un grupo es eliminatorio, no superar su nota mínima suspende todo el examen. Con grupos
          activos, la configuración de corrección/puntuación de abajo se ignora y cada pregunta
          debe asignarse a un grupo.
        </p>
        <div className="flex flex-col gap-4">
          {groups.map((g, gi) => (
            <div key={g.position} className="border border-[#444] rounded-lg p-3 bg-[#23262c]">
              <div className="flex flex-wrap gap-3 items-end">
                <label className="font-bold text-brand-pink">
                  Nombre:
                  <input
                    type="text"
                    value={g.name}
                    onInput={(e) => updateGroup(gi, { name: (e.target as HTMLInputElement).value })}
                    placeholder="Teórico"
                    className="block w-44 mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
                    disabled={isBusy}
                  />
                </label>
                <label className="font-bold text-brand-pink">
                  Valoración (puntos):
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    value={g.max_score}
                    onInput={(e) => updateGroup(gi, { max_score: parseFloat((e.target as HTMLInputElement).value) || 0 })}
                    className="block w-28 mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
                    disabled={isBusy}
                  />
                </label>
                <label className="font-bold text-brand-pink">
                  Resta por fallo:
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    value={g.points_per_wrong}
                    onInput={(e) => updateGroup(gi, { points_per_wrong: parseFloat((e.target as HTMLInputElement).value) || 0 })}
                    className="block w-28 mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
                    disabled={isBusy}
                  />
                </label>
                <label className="font-bold text-brand-pink">
                  Nota mínima:
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    value={g.min_passing_score ?? ''}
                    onInput={(e) => {
                      const raw = (e.target as HTMLInputElement).value;
                      updateGroup(gi, { min_passing_score: raw === '' ? null : parseFloat(raw) });
                    }}
                    placeholder="(sin mínimo)"
                    className="block w-28 mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
                    disabled={isBusy}
                  />
                </label>
                <label className="flex items-center gap-2 text-brand-pink font-bold">
                  <input
                    type="checkbox"
                    checked={g.eliminatory}
                    onChange={(e) => updateGroup(gi, { eliminatory: e.currentTarget.checked })}
                    disabled={isBusy}
                  />
                  Eliminatorio
                </label>
                <button
                  type="button"
                  onClick={() => handleRemoveGroup(gi)}
                  className="bg-red-600 text-white border-none rounded px-3 py-2 cursor-pointer disabled:opacity-60"
                  disabled={isBusy}
                >
                  Eliminar grupo
                </button>
              </div>
            </div>
          ))}
          <button
            type="button"
            onClick={handleAddGroup}
            className="self-start bg-brand-blue text-white border-none rounded px-4 py-2 cursor-pointer hover:bg-brand-blue/80 disabled:opacity-60"
            disabled={isBusy}
          >
            + Añadir grupo
          </button>
        </div>
      </fieldset>

      {!grouped && (
      <>
      <fieldset className="mb-6 border border-[#444] rounded-lg p-4" disabled={isBusy}>
        <legend className="font-bold text-brand-pink px-2">Configuración de corrección</legend>
        <div className="flex flex-col gap-4">
          <label className="flex items-center gap-2 flex-wrap">
            <span className="font-bold text-brand-pink">Modo de puntuación:</span>
            <select
              value={scoringMode}
              onChange={(e) => setScoringMode(e.currentTarget.value as 'legacy' | 'absolute')}
              className="px-3 py-1 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
              disabled={isBusy}
            >
              <option value="legacy">Clásico (base + penalización fraccional)</option>
              <option value="absolute">Absoluto (puntos por acierto/fallo)</option>
            </select>
          </label>

          {scoringMode === 'legacy' && (
            <>
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
                <label className="flex items-center gap-2 ml-6 flex-wrap">
                  <span className="text-white">Cantidad a restar por fallo:</span>
                  <select
                    value={penaltyValue}
                    onChange={(e) => setPenaltyValue(parseFloat(e.currentTarget.value))}
                    className="px-3 py-1 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
                    disabled={isBusy}
                  >
                    <option value={0}>0</option>
                    <option value={0.1}>0.1</option>
                    <option value={0.25}>0.25</option>
                    <option value={0.33333333}>0.33</option>
                    <option value={0.5}>0.5</option>
                  </select>
                  <span className="text-xs text-gray-400">(fracción de un acierto)</span>
                </label>
              )}
            </>
          )}

          {scoringMode === 'absolute' && (
            <div className="ml-6 flex flex-col gap-3">
              <label className="flex items-center gap-2 flex-wrap">
                <span className="text-white w-52">Puntos por acierto:</span>
                <input
                  type="number"
                  step="0.01"
                  min="0"
                  value={pointsPerCorrect}
                  onInput={(e) => setPointsPerCorrect(parseFloat((e.target as HTMLInputElement).value) || 0)}
                  className="w-32 px-3 py-1 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
                  disabled={isBusy}
                />
              </label>
              <label className="flex items-center gap-2 flex-wrap">
                <span className="text-white w-52">Puntos a restar por fallo:</span>
                <input
                  type="number"
                  step="0.01"
                  min="0"
                  value={pointsPerWrong}
                  onInput={(e) => setPointsPerWrong(parseFloat((e.target as HTMLInputElement).value) || 0)}
                  className="w-32 px-3 py-1 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
                  disabled={isBusy}
                />
              </label>
              <p className="text-xs text-gray-400">
                Valores positivos. Cada acierto suma esos puntos y cada fallo los resta; la nota nunca baja de 0.
                En este modo la "Puntuación máxima" de abajo se ignora: el máximo es puntos por acierto × nº de preguntas.
              </p>
            </div>
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
      </>
      )}

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

      <fieldset className="mb-6 border border-[#444] rounded-lg p-4" disabled={isBusy}>
        <legend className="font-bold text-brand-pink px-2">Ponderacion nota final</legend>
        <p className="text-xs text-gray-400 mb-4">
          Configura el peso del examen y meritos en la nota final ponderada.
        </p>
        <label className="flex items-center gap-2 mb-4 cursor-pointer">
          <input
            type="checkbox"
            checked={skipWeights}
            onChange={(e) => setSkipWeights((e.target as HTMLInputElement).checked)}
            disabled={isBusy}
            className="accent-brand-pink"
          />
          <span className="text-brand-pink font-bold">Ignorar pesos (sumar directamente)</span>
          <span className="text-xs text-gray-400">Nota ponderada = nota examen + meritos</span>
        </label>
        {!skipWeights && (
          <div className="flex flex-col gap-4 sm:flex-row sm:gap-6 mb-4">
            <label className="block text-brand-pink font-bold flex-1">
              Peso del examen:
              <input
                type="number"
                step="0.01"
                min="0"
                max="1"
                value={examWeight}
                onInput={(e) => {
                  const v = parseFloat((e.target as HTMLInputElement).value);
                  if (!isNaN(v)) setExamWeight(Math.min(1, Math.max(0, v)));
                }}
                className="w-full mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
                disabled={isBusy}
              />
            </label>
            <div className="flex-1">
              <span className="block text-brand-pink font-bold">Peso de meritos:</span>
              <span className="block mt-1 px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-gray-300">
                {(1 - examWeight).toFixed(2)}
              </span>
            </div>
          </div>
        )}
        <label className="flex items-center gap-2 mt-2 cursor-pointer">
          <input
            type="checkbox"
            checked={displayWeightOverride}
            onChange={(e) => {
              setDisplayWeightOverride((e.target as HTMLInputElement).checked);
              if ((e.target as HTMLInputElement).checked) setDisplayExamWeight(examWeight);
            }}
            disabled={isBusy}
            className="accent-brand-pink"
          />
          <span className="text-brand-pink font-bold">Mostrar pesos diferentes a los alumnos</span>
        </label>
        {displayWeightOverride && (
          <div className="flex flex-col gap-4 sm:flex-row sm:gap-6 mt-3 ml-6 p-3 rounded border border-purple-500/30 bg-[#2a1f2a]/30">
            <label className="block text-purple-300 font-bold flex-1">
              Peso examen (visible):
              <input
                type="number"
                step="0.01"
                min="0"
                max="1"
                value={displayExamWeight}
                onInput={(e) => {
                  const v = parseFloat((e.target as HTMLInputElement).value);
                  if (!isNaN(v)) setDisplayExamWeight(Math.min(1, Math.max(0, v)));
                }}
                className="w-full mt-1 px-3 py-2 rounded border border-purple-500/30 bg-[#2a2d33] text-white focus:outline-none focus:border-purple-400"
                disabled={isBusy}
              />
            </label>
            <div className="flex-1">
              <span className="block text-purple-300 font-bold">Peso meritos (visible):</span>
              <span className="block mt-1 px-3 py-2 rounded border border-purple-500/30 bg-[#1f2229] text-gray-300">
                {(1 - displayExamWeight).toFixed(2)}
              </span>
            </div>
            <p className="text-xs text-purple-300/60 sm:col-span-2">
              Los alumnos veran estos pesos, pero el calculo real {skipWeights ? 'suma directamente' : 'usa los pesos configurados arriba'}.
            </p>
          </div>
        )}

        <label className="block text-brand-pink font-bold mt-4">
          Tope de meritos:
          <input
            type="number"
            step="0.01"
            min="0"
            value={maxMerits}
            onInput={(e) => {
              const v = parseFloat((e.target as HTMLInputElement).value);
              if (!isNaN(v) && v > 0) setMaxMerits(v);
            }}
            className="w-full mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
            disabled={isBusy}
          />
          <span className="text-xs text-gray-400 block mt-1">
            Valor maximo que un alumno puede introducir como meritos. Por defecto 100.
          </span>
        </label>
      </fieldset>

      <fieldset className="mb-6 border border-[#444] rounded-lg p-4" disabled={isBusy}>
        <legend className="font-bold text-brand-pink px-2">Bases del sorteo</legend>
        <label className="flex items-center gap-2 mb-4 cursor-pointer">
          <input
            type="checkbox"
            checked={raffleEnabled}
            onChange={(e) => setRaffleEnabled((e.target as HTMLInputElement).checked)}
            disabled={isBusy}
            className="accent-brand-pink"
          />
          <span className="text-brand-pink font-bold">Añadir bases de sorteo</span>
          <span className="text-xs text-gray-400">El alumno deberá aceptarlas para entregar.</span>
        </label>
        {raffleEnabled && (
          <label className="block text-brand-pink font-bold">
            Texto de las bases:
            <textarea
              value={raffleTerms}
              onInput={(e) => setRaffleTerms((e.target as HTMLTextAreaElement).value)}
              rows={8}
              className="w-full mt-1 px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue"
              disabled={isBusy}
            />
          </label>
        )}
      </fieldset>

      <fieldset className="border border-[#444] p-4 rounded-lg" disabled={isBusy}>
        <legend className="font-bold text-xl mb-4 text-brand-pink">Preguntas</legend>

        <p className="text-xs text-gray-400 mb-4">
          El campo <strong>Nº</strong> define la posición final de cada pregunta para el
          alumno. Puedes editarlos libremente (incluyendo reservas intercaladas entre
          preguntas normales); las tarjetas no se reordenan mientras escribes para no
          perder el foco. Al guardar, los números deben formar la secuencia 1..{questions.length}.
        </p>

        <section className="mt-2">
          {questionEntries.map((entry) => renderQuestionCard(entry))}
        </section>

        <button
          type="button"
          onClick={handleAddQuestion}
          className="bg-brand-blue text-white border-none rounded px-4 py-2 cursor-pointer mt-6 hover:bg-brand-blue/80"
          disabled={isBusy}
        >
          Anadir pregunta
        </button>
      </fieldset>

      <div className="sticky bottom-0 z-10 bg-[#14161d] border-t border-brand-blue-soft py-4 -mx-4 px-4 mt-8 shadow-[0_-4px_12px_rgba(0,0,0,0.3)]">
        <button
          type="submit"
          disabled={isBusy}
          className="w-full py-3 text-lg font-bold cursor-pointer rounded-md bg-brand-blue hover:bg-brand-blue/80 transition-colors disabled:opacity-70 disabled:cursor-not-allowed"
        >
          {loading
            ? examToEdit
              ? 'Guardando...'
              : 'Creando...'
            : examToEdit
            ? 'Guardar cambios'
            : 'Crear examen'}
        </button>
      </div>

      <AssociateExamsModal
        isOpen={associateModalOpen}
        exams={otherExams}
        selectedIds={associatedExamIds}
        onSave={setAssociatedExamIds}
        onClose={() => setAssociateModalOpen(false)}
      />
    </form>
  );
}

// FeedbackVideoKeyField — inline component for setting/clearing feedback_video_key
// on an existing question via PATCH /admin/questions/{id}.
function FeedbackVideoKeyField({ questionId, token }: { questionId: number; token: string }) {
  const [key, setKey] = useState('');
  const [status, setStatus] = useState<'idle' | 'saving' | 'ok' | 'error'>('idle');
  const rootRef = useRef<HTMLDivElement>(null);

  // Fetch the saved key only when this field scrolls into view, so opening an
  // exam with many questions doesn't fire one request per question at once.
  useEffect(() => {
    const el = rootRef.current;
    if (!el) return;
    let done = false;
    const load = () => {
      if (done) return;
      done = true;
      getQuestionFeedbackVideo(questionId, token)
        .then((k) => { if (k) setKey(k); })
        .catch(() => {});
    };
    const observer = new IntersectionObserver((entries) => {
      if (entries.some((e) => e.isIntersecting)) {
        load();
        observer.disconnect();
      }
    }, { rootMargin: '200px' });
    observer.observe(el);
    return () => observer.disconnect();
  }, [questionId, token]);

  async function save(value: string | null) {
    setStatus('saving');
    try {
      await patchQuestionFeedbackVideo(questionId, value, token);
      setStatus('ok');
      if (value === null) setKey('');
      setTimeout(() => setStatus('idle'), 2000);
    } catch {
      setStatus('error');
      setTimeout(() => setStatus('idle'), 3000);
    }
  }

  return (
    <div ref={rootRef} className="mt-4 pt-3 border-t border-[#333]">
      <p className="text-xs font-semibold text-gray-400 mb-1">Vídeo de feedback (key S3)</p>
      <div className="flex gap-2 items-center flex-wrap">
        <input
          type="text"
          value={key}
          placeholder="examen-2025/pregunta-42"
          onInput={(e) => setKey((e.target as HTMLInputElement).value.trim())}
          className="flex-1 min-w-0 px-3 py-1.5 rounded border border-[#444] bg-[#1f2229] text-white text-sm focus:outline-none focus:border-brand-blue"
        />
        <button
          type="button"
          disabled={status === 'saving' || !key}
          onClick={() => save(key)}
          className="px-3 py-1.5 rounded bg-brand-blue text-white text-sm font-semibold disabled:opacity-50 cursor-pointer"
        >
          {status === 'saving' ? '…' : 'Guardar'}
        </button>
        <button
          type="button"
          disabled={status === 'saving'}
          onClick={() => save(null)}
          className="px-3 py-1.5 rounded border border-red-500/50 text-red-400 text-sm font-semibold disabled:opacity-50 cursor-pointer"
        >
          Borrar
        </button>
        {status === 'ok' && <span className="text-xs text-green-400">✓ Guardado</span>}
        {status === 'error' && <span className="text-xs text-red-400">Error al guardar</span>}
      </div>
    </div>
  );
}
