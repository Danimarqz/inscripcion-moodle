import type { JSX } from 'preact';
import { useEffect, useMemo, useState } from 'preact/hooks';

import type {
  Answer,
  AnswerReview,
  ExamOut,
  ExamResultPayload,
  ExamSubmissionPayload,
  Question,
} from '../types/exam';
import { checkSubmission, submitExam } from '../services/examService';
import { useExamQuestions } from '../hooks/useExamQuestions';
import { useExamUi } from '../hooks/useExamUi';
import { isValidEmail, normalizeDni, roundToTwoDecimals, validateDniNie } from '../utils/validation';

interface ExamPageProps {
  examId: number;
  examName: string;
  showScore: boolean;
  showPercentile: boolean;
  showScoreFull: boolean;
  validatedTribunal?: boolean;
}

const ANSWER_OPTIONS = ['A', 'B', 'C', 'D'];

type ComposeMessageParams = {
  baseMessage?: string | null;
  showScore: boolean;
  showPercentile: boolean;
  showScoreFull: boolean;
  score: number | null;
  percentile: number | null;
  position: number | null;
  totalSubmissions: number | null;
  correctAnswers: number | null;
  totalQuestions: number | null;
};

function composeResultMessage({
  baseMessage,
  showScore,
  showPercentile,
  showScoreFull,
  score,
  percentile,
  position,
  totalSubmissions,
  correctAnswers,
  totalQuestions,
}: ComposeMessageParams): string {
  const parts: string[] = [];
  if (baseMessage) {
    parts.push(baseMessage.trim());
  }
  if (showScore && score !== null) {
    parts.push(`Tu puntuación es ${score}`);
  }
  if (showScoreFull && correctAnswers !== null && totalQuestions !== null) {
    parts.push(`Has acertado ${correctAnswers} de ${totalQuestions} preguntas`);
  }
  if (showPercentile && percentile !== null) {
    let percentileMessage = `Tienes el percentil ${percentile}`;
    if (position !== null && totalSubmissions !== null) {
      percentileMessage += `, estás en la posición ${position} de ${totalSubmissions}`;
    }
    parts.push(percentileMessage);
  }

  const result = parts.filter(Boolean).join('. ').trim();
  return result || 'Tu entrega ha sido registrada correctamente';
}

export default function ExamPage({
  examId,
  examName,
  showScore,
  showPercentile,
  showScoreFull,
  validatedTribunal = false,
}: ExamPageProps) {
  const [studentName, setStudentName] = useState('');
  const [studentSurname, setStudentSurname] = useState('');
  const [email, setEmail] = useState('');
  const [dni, setDni] = useState('');
  const [acceptsMarketing, setAcceptsMarketing] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [autoCheckDisabled, setAutoCheckDisabled] = useState(false);
  const [userAnswers, setUserAnswers] = useState<Record<number, string>>({});

  const [examUiState, dispatchExamUi] = useExamUi();
  const {
    checking,
    hasPreviousSubmission,
    submissionMessage,
    resultError,
    score: latestScore,
    percentile: latestPercentile,
    position,
    totalSubmissions,
    correctAnswers,
    totalQuestions,
    answersReview,
  } = examUiState;

  const { questions, loading, error: questionsError } = useExamQuestions(examId);
  const allowResultPreview =
    showScore || showPercentile || showScoreFull || validatedTribunal;

  useEffect(() => {
    dispatchExamUi({ type: 'RESET' });
    setAutoCheckDisabled(false);
    setAcceptsMarketing(false);
    setUserAnswers({});
  }, [examId, dispatchExamUi]);

  const questionEntries = useMemo(
    () => questions.map((question, index) => ({ index, question })),
    [questions],
  );
  const activeEntries = questionEntries.filter(({ question }) => question.is_active !== false);
  const reserveEntries = questionEntries.filter(({ question }) => question.is_active === false);

  function setAnswerForQuestion(questionId: number, option: string) {
    const normalizedOption = option.toUpperCase();
    setUserAnswers((prev) => ({
      ...prev,
      [questionId]: normalizedOption,
    }));
  }

  function clearAnswerForQuestion(questionId: number) {
    setUserAnswers((prev) => {
      if (!(questionId in prev)) {
        return prev;
      }
      const next = { ...prev };
      delete next[questionId];
      return next;
    });
  }

  function buildResultPayload(result: ExamOut, context: 'check' | 'submit'): ExamResultPayload {
    const baseMessage =
      result.message ??
      (context === 'check'
        ? 'Ya has entregado este examen anteriormente'
        : 'Tu entrega ha sido registrada correctamente');

    const nextScore =
      showScore && typeof result.score === 'number' ? roundToTwoDecimals(result.score) : null;
    const nextPercentile =
      showPercentile && typeof result.percentile === 'number'
        ? roundToTwoDecimals(result.percentile)
        : null;
    const rawPosition = typeof result.position === 'number' ? result.position : null;
    const rawTotalSubmissions =
      typeof result.total_submissions === 'number' ? result.total_submissions : null;
    const rawCorrectAnswers =
      typeof result.correct_answers === 'number' ? result.correct_answers : null;
    const rawTotalQuestions =
      typeof result.total_questions === 'number' ? result.total_questions : null;
    const review =
      Array.isArray(result.answers_review) && result.answers_review.length > 0
        ? (result.answers_review as AnswerReview[])
        : null;

    const nextPosition = showPercentile ? rawPosition : null;
    const nextTotalSubmissions = showPercentile ? rawTotalSubmissions : null;
    const nextCorrectAnswers = showScoreFull ? rawCorrectAnswers : null;
    const nextTotalQuestions = showScoreFull ? rawTotalQuestions : null;

    const message = composeResultMessage({
      baseMessage,
      showScore,
      showPercentile,
      showScoreFull,
      score: nextScore,
      percentile: nextPercentile,
      position: nextPosition,
      totalSubmissions: nextTotalSubmissions,
      correctAnswers: nextCorrectAnswers,
      totalQuestions: nextTotalQuestions,
    });

    return {
      score: nextScore,
      percentile: nextPercentile,
      position: nextPosition,
      totalSubmissions: nextTotalSubmissions,
      correctAnswers: nextCorrectAnswers,
      totalQuestions: nextTotalQuestions,
      message,
      answersReview: review,
    };
  }

  async function checkUserSubmission(force = false) {
    if (!allowResultPreview) return;
    if (!force && autoCheckDisabled) return;
    if (checking) return;

    const normalizedEmail = email.trim().toLowerCase();
    const normalizedDni = normalizeDni(dni);

    if (!isValidEmail(normalizedEmail) || !validateDniNie(normalizedDni)) return;

    dispatchExamUi({ type: 'CHECK_START' });

    try {
      const data: ExamOut = await checkSubmission({ email: normalizedEmail, dni: normalizedDni, exam_id: examId });
      const payload = buildResultPayload(data, 'check');
      dispatchExamUi({ type: 'CHECK_SUCCESS', payload });
      setAutoCheckDisabled(true);
    } catch (error) {
      const message = (error as Error).message;
      if (message && message !== 'Submission not found') {
        dispatchExamUi({ type: 'CHECK_ERROR', payload: message });
      } else {
        dispatchExamUi({ type: 'CHECK_ERROR' });
      }
    }
  }

  const handleCheckSubmissionBlur = (_event: JSX.TargetedFocusEvent<HTMLInputElement>) => {
    void checkUserSubmission();
  };

  function handleManualCheck(event: Event) {
    event.preventDefault();
    void checkUserSubmission(true);
  }

  const handleSubmit = async (event: Event) => {
    event.preventDefault();
    setFormError(null);

    const trimmedName = studentName.trim();
    const trimmedSurname = studentSurname.trim();

    if (!trimmedName) {
      setFormError('El nombre es obligatorio.');
      return;
    }
    if (!trimmedSurname) {
      setFormError('Los apellidos son obligatorios.');
      return;
    }

    const form = event.currentTarget as HTMLFormElement;
    const formData = new FormData(form);

    const emailRaw = (formData.get('email') as string).trim().toLowerCase();
    const dniRaw = normalizeDni(formData.get('dni') as string);

    if (!isValidEmail(emailRaw)) {
      setFormError('Por favor, introduce un email valido.');
      return;
    }
    if (!validateDniNie(dniRaw)) {
      setFormError('Por favor, introduce un DNI o NIE valido.');
      return;
    }
    if (!acceptsMarketing) {
      setFormError('Debes aceptar el uso de tus datos para comunicaciones necesarias.');
      return;
    }

    const answers: Answer[] = Object.entries(userAnswers).reduce<Answer[]>((acc, [questionId, value]) => {
      const numericId = Number(questionId);
      if (Number.isFinite(numericId) && value) {
        acc.push({ question_id: numericId, answer: value });
      }
      return acc;
    }, []);

    const payload: ExamSubmissionPayload = {
      email: emailRaw,
      dni: dniRaw,
      name: trimmedName,
      surname: trimmedSurname,
      exam_id: examId,
      answers,
      accepts_marketing: acceptsMarketing,
    };

    try {
      const result = await submitExam(payload);
      const payloadResult = buildResultPayload(result, 'submit');

      dispatchExamUi({
        type: 'SUBMIT_SUCCESS',
        payload: payloadResult,
      });
      setAutoCheckDisabled(true);
      setStudentName('');
      setStudentSurname('');
      setEmail('');
      setDni('');
      setAcceptsMarketing(false);
      setUserAnswers({});
    } catch (error) {
      const message = (error as Error).message || 'Error al enviar el examen';
      dispatchExamUi({ type: 'SUBMIT_ERROR', payload: message });
      setFormError(message);
    }
  };

  function renderSubmissionSummary(message: string, review: AnswerReview[] | null) {
    const trimmedMessage = message.trim();
    if (!trimmedMessage) return null;
    const sentenceMatch = trimmedMessage.match(/.*?[.!?](?:\s|$)/);
    const primaryMessage = (sentenceMatch ? sentenceMatch[0] : trimmedMessage).trim();
    return (
      <div className="mt-6 rounded-2xl border border-green-500/60 bg-green-500/5 p-6 shadow-[0_10px_25px_rgba(16,185,129,0.15)] text-left">
        <p className="text-green-200 text-lg font-semibold mb-2">
          {primaryMessage ?? trimmedMessage}
        </p>

        <div className="flex flex-wrap gap-4">
          {showScore && typeof latestScore === 'number' && (
            <div className="flex-1 min-w-[180px] rounded-xl bg-[#1f2a24] border border-green-500/30 p-4">
              <p className="text-xs uppercase tracking-widest text-green-400/80">Puntuación</p>
              <p className="text-2xl font-bold text-green-200">{latestScore}</p>
            </div>
          )}
          {showScoreFull && typeof correctAnswers === 'number' && typeof totalQuestions === 'number' && (
            <div className="flex-1 min-w-[180px] rounded-xl bg-[#1f252a] border border-teal-500/30 p-4">
              <p className="text-xs uppercase tracking-widest text-teal-300/80">Aciertos</p>
              <p className="text-2xl font-bold text-teal-200">
                {correctAnswers} <span className="text-base text-teal-200/70">de</span> {totalQuestions}
              </p>
            </div>
          )}
          {showPercentile && typeof latestPercentile === 'number' && (
            <div className="flex-1 min-w-[180px] rounded-xl bg-[#1f2330] border border-indigo-500/30 p-4">
              <p className="text-xs uppercase tracking-widest text-indigo-300/80">Percentil</p>
              <p className="text-2xl font-bold text-indigo-200">{latestPercentile}</p>
              {typeof position === 'number' && typeof totalSubmissions === 'number' && (
                <p className="text-sm text-indigo-200/70 mt-1">
                  Posición {position} de {totalSubmissions}
                </p>
              )}
            </div>
          )}
        </div>
        {Array.isArray(review) && review.length > 0 && (
          <div className="mt-6">
            <h3 className="text-base font-semibold text-brand-yellow mb-3">
              Detalle de respuestas
            </h3>
            <div className="overflow-x-auto rounded-2xl border border-brand-blue-soft bg-[#12141b]">
              <table className="min-w-full text-sm">
                <thead className="text-xs uppercase tracking-[0.35em] text-brand-yellow/80">
                  <tr>
                    <th className="px-4 py-2 text-left">Pregunta</th>
                    <th className="px-4 py-2 text-left">Tu respuesta</th>
                    <th className="px-4 py-2 text-left">Respuesta correcta</th>
                    <th className="px-4 py-2 text-left">Estado</th>
                  </tr>
                </thead>
                <tbody>
                  {review.map((item, index) => {
                    const questionNumber = item.question_label ?? index + 1;
                    const selected = item.selected_option ?? 'Sin marcar';
                    const correct = item.correct_option ?? '—';
                    const isAnswered = Boolean(item.selected_option);
                    const statusLabel = item.is_correct
                      ? 'Correcta'
                      : isAnswered
                        ? 'Incorrecta'
                        : 'Sin responder';
                    const rowClass = item.is_correct
                      ? 'bg-green-500/5'
                      : isAnswered
                        ? 'bg-brand-pink-soft'
                        : '';
                    const statusClass = item.is_correct
                      ? 'text-green-300'
                      : isAnswered
                        ? 'text-brand-pink'
                        : 'text-gray-400';
                    return (
                      <tr key={`${item.question_id}-${index}`} className={rowClass}>
                        <td className="px-4 py-2 font-semibold text-white">
                          Pregunta {questionNumber}
                        </td>
                        <td className="px-4 py-2">
                          <span
                            className={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-semibold ${
                              item.is_correct
                                ? 'border-green-500/40 text-green-300'
                                : isAnswered
                                  ? 'border-brand-pink-soft text-brand-pink'
                                  : 'border-gray-600 text-gray-400'
                            }`}
                          >
                            {selected}
                          </span>
                        </td>
                        <td className="px-4 py-2 text-brand-yellow font-semibold">{correct}</td>
                        <td className={`px-4 py-2 text-sm font-semibold ${statusClass}`}>{statusLabel}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    );
  }

  function renderQuestionCard(
    entry: { index: number; question: Question },
    position: number,
    isReserve: boolean,
  ) {
    const { question, index: originalIndex } = entry;
    const questionId = question.id;
    if (typeof questionId !== 'number') {
      return null;
    }
    const isCancelled = question.is_cancelled === true;
    const displayName = question.name ?? position;
    const label = `${isReserve ? 'Reserva' : 'Pregunta'} ${displayName}`;
    const key = `${questionId}-${isReserve ? 'reserve' : 'active'}-${originalIndex}`;
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
    const selectedOption = userAnswers[questionId] ?? null;

    return (
      <div
        className="bg-[#2a2d33] p-6 mb-4 mt-4 rounded-lg shadow-lg border border-transparent hover:border-brand-blue/40 transition-colors"
        key={key}
      >
        <div className="flex items-center justify-between mb-4">
          <p className="font-bold text-lg text-brand-blue text-opacity-80">{label}</p>
          <span className={`text-xs font-semibold px-2 py-1 rounded ${badgeConfig.className}`}>
            {badgeConfig.text}
          </span>
        </div>
        <ul className="list-none p-0 m-0 flex flex-wrap gap-3">
          {ANSWER_OPTIONS.map((optionChar) => {
            const isSelected = selectedOption === optionChar;
            return (
              <li key={`${questionId}-${optionChar}`}>
                <button
                  type="button"
                  onClick={() => {
                    if (isCancelled) return;
                    if (isSelected) {
                      clearAnswerForQuestion(questionId);
                    } else {
                      setAnswerForQuestion(questionId, optionChar);
                    }
                  }}
                  disabled={isCancelled}
                  aria-pressed={isSelected}
                  className={`w-16 text-center px-4 py-2 text-lg font-semibold rounded-md border transition-all duration-200 ${
                    isCancelled
                      ? 'border-gray-600 text-gray-600 cursor-not-allowed'
                      : isSelected
                        ? 'border-brand-yellow bg-brand-yellow text-dark-200 shadow-lg cursor-pointer'
                        : 'border-[#555] text-gray-200 hover:border-brand-blue hover:text-brand-yellow cursor-pointer'
                  }`}
                >
                  {optionChar}
                </button>
              </li>
            );
          })}
        </ul>
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <p className="text-xs text-gray-500">
            Usa los botones para marcar tu opción; si pulsas de nuevo o en &ldquo;Borrar respuesta&rdquo; la pregunta
            queda en blanco.
          </p>
          <button
            type="button"
            onClick={() => clearAnswerForQuestion(questionId)}
            disabled={!selectedOption}
            className="text-xs font-semibold px-3 py-1 rounded border border-brand-blue-soft text-brand-blue hover:bg-brand-blue-soft disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
          >
            Borrar respuesta
          </button>
        </div>
      </div>
    );
  }

  if (loading) {
    return (
      <main>
        <p className="info-message">Cargando preguntas...</p>
      </main>
    );
  }

  if (questionsError)
    return (
      <main>
        <p className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mt-6">{questionsError}</p>
      </main>
    );

  if (questions.length === 0 || activeEntries.length === 0)
    return (
      <main>
        <p className="info-message">No hay preguntas activas disponibles para este examen.</p>
      </main>
    );

  const resultsSummary = submissionMessage ? renderSubmissionSummary(submissionMessage, answersReview) : null;

  return (
    <main>
      <a
        href="/"
        className="inline-block mb-6 px-4 py-2 font-bold text-brand-pink border border-brand-pink rounded-md no-underline transition-colors duration-300 ease-in-out hover:text-brand-yellow hover:border-brand-yellow"
      >
        &larr; Volver a la seleccion de examen
      </a>
      <h1 className="text-5xl font-extrabold leading-tight text-center mb-12 text-transparent bg-gradient-to-r from-brand-pink via-brand-yellow to-brand-blue bg-clip-text drop-shadow-[0_20px_45px_rgba(15,153,188,0.35)]">
        {examName}
      </h1>

      {allowResultPreview && (
        <section className="mb-10 rounded-2xl border border-brand-blue-soft bg-brand-blue-soft p-6 shadow-[0_10px_30px_rgba(15,153,188,0.2)]">
          <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.4em] text-brand-yellow">Consulta rápida</p>
              <h2 className="text-2xl font-bold text-brand-pink">Consultar mi nota</h2>
              <p className="text-sm text-brand-blue text-opacity-80 mt-1">
                Introduce el mismo email y DNI/NIE con el que enviaste el intento para ver tu resultado.
              </p>
              {validatedTribunal && (
                <p className="text-xs text-brand-yellow mt-2">
                  Este examen est&aacute; validado por tribunal: al consultar ver&aacute;s tus respuestas comparadas con las correctas.
                </p>
              )}
            </div>
          </div>
          <form
            className="flex flex-col gap-4 md:flex-row md:items-end"
            onSubmit={handleManualCheck}
            noValidate
          >
            <label className="flex-1 text-sm font-semibold text-brand-pink">
              Email
              <input
                type="email"
                value={email}
                onInput={(event) => setEmail((event.target as HTMLInputElement).value)}
                className="mt-1 w-full rounded border border-[#5a4a7a] bg-[#1d1f27] px-3 py-2 text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/40"
                placeholder="tu@email.com"
                required
              />
            </label>
            <label className="flex-1 text-sm font-semibold text-brand-pink">
              DNI / NIE
              <input
                type="text"
                value={dni}
                onInput={(event) => setDni(normalizeDni((event.target as HTMLInputElement).value))}
                className="mt-1 w-full rounded border border-[#5a4a7a] bg-[#1d1f27] px-3 py-2 text-white uppercase focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/40"
                placeholder="00000000A"
                required
              />
            </label>
            <button
              type="submit"
              className="btn-brand w-full md:w-auto px-6"
              disabled={checking}
            >
              {checking ? 'Consultando...' : 'Consultar mi nota'}
            </button>
          </form>
        </section>
      )}
      {allowResultPreview && resultsSummary}

      <form id="exam-form" onSubmit={handleSubmit} noValidate>
        <div className="mb-6">
          <label htmlFor="name" className="block font-bold text-brand-pink mb-2">Nombre:</label>
          <input
            type="text"
            id="name"
            name="name"
            required
            value={studentName}
            onInput={(event) => setStudentName((event.target as HTMLInputElement).value)}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
          />
        </div>

        <div className="mb-6">
          <label htmlFor="surname" className="block font-bold text-brand-pink mb-2">Apellidos:</label>
          <input
            type="text"
            id="surname"
            name="surname"
            required
            value={studentSurname}
            onInput={(event) => setStudentSurname((event.target as HTMLInputElement).value)}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
          />
        </div>

        <div className="mb-6">
          <label htmlFor="email" className="block font-bold text-brand-pink mb-2">Email:</label>
          <input
            type="email"
            id="email"
            name="email"
            required
            value={email}
            onInput={(event) => setEmail((event.target as HTMLInputElement).value)}
            onBlur={handleCheckSubmissionBlur}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
          />
        </div>

        <div className="mb-6">
          <label htmlFor="dni" className="block font-bold text-brand-pink mb-2">DNI/NIE:</label>
          <input
            type="text"
            id="dni"
            name="dni"
            required
            value={dni}
            onInput={(event) => setDni(normalizeDni((event.target as HTMLInputElement).value))}
            onBlur={handleCheckSubmissionBlur}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
          />
        </div>

        {checking && (
          <p className="text-center text-brand-blue bg-brand-blue/10 border border-brand-blue/50 p-4 rounded-md mt-6">
            Comprobando si ya has entregado el examen...
          </p>
        )}
        {!allowResultPreview && resultsSummary}
        {resultError && (
          <p className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mt-6">
            {resultError}
          </p>
        )}

        {!hasPreviousSubmission && (
          <>
            <section>
              <h2 className="text-2xl font-semibold text-brand-pink mt-8">Preguntas activas</h2>
              <p className="text-sm text-gray-400 mt-1">Responde todas las preguntas activas; puntuan en la nota final.</p>
              {activeEntries.map((entry, index) => renderQuestionCard(entry, index + 1, false))}
            </section>

            {reserveEntries.length > 0 && (
              <section className="mt-8">
                <h2 className="text-2xl font-semibold text-brand-pink">Preguntas de reserva</h2>
                {reserveEntries.map((entry, index) => renderQuestionCard(entry, activeEntries.length + index + 1, true))}
              </section>
            )}

            <div className="mt-8 space-top-3 text-sm">
              <label htmlFor="accepts_marketing" className="flex items-start gap-3 text-gray-200">
                <input
                  type="checkbox"
                  id="accepts_marketing"
                  name="accepts_marketing"
                  required
                  checked={acceptsMarketing}
                  onChange={(event) => setAcceptsMarketing(event.currentTarget.checked)}
                  className="mt-1 h-4 w-4 rounded border border-[#555] bg-[#2a2d33] text-brand-pink focus:ring-2 focus:ring-brand-yellow/60"
                />
                <span className="text-gray-300">
                  Acepto recibir por email recordatorios y novedades sobre nuevas oposiciones.
                </span>
              </label>
              <p className="text-xs leading-relaxed text-gray-400">
                Al entregar confirmas que utilizaremos tus datos solo para corregir tu simulacro y gestionar el servicio;
                no usamos cookies de seguimiento. Consulta la{' '}
                <a
                  href="/politica-de-privacidad"
                  className="text-brand-pink underline decoration-dotted hover:text-brand-yellow"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Política de privacidad
                </a>
                .
              </p>
            </div>
          </>
        )}

        {!hasPreviousSubmission && (
          <button
            type="submit"
            className="btn-brand w-full text-lg mt-4"
          >
            Entregar Examen
          </button>
        )}
        {formError && (
          <p className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mt-6">{formError}</p>
        )}
      </form>
    </main>
  );
}
