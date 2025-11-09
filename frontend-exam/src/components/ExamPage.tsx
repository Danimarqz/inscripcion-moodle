import { useEffect, useMemo, useState } from 'preact/hooks';

import type { Answer, ExamOut, ExamResultPayload, ExamSubmissionPayload, Question } from '../types/exam';
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
}: ExamPageProps) {
  const [studentName, setStudentName] = useState('');
  const [studentSurname, setStudentSurname] = useState('');
  const [email, setEmail] = useState('');
  const [dni, setDni] = useState('');
  const [acceptsMarketing, setAcceptsMarketing] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [autoCheckDisabled, setAutoCheckDisabled] = useState(false);

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
  } = examUiState;

  const { questions, loading, error: questionsError } = useExamQuestions(examId);
  const allowResultPreview = showScore || showPercentile || showScoreFull;

  useEffect(() => {
    dispatchExamUi({ type: 'RESET' });
    setAutoCheckDisabled(false);
    setAcceptsMarketing(false);
  }, [examId, dispatchExamUi]);

  const questionEntries = useMemo(
    () => questions.map((question, index) => ({ index, question })),
    [questions],
  );
  const activeEntries = questionEntries.filter(({ question }) => question.is_active !== false);
  const reserveEntries = questionEntries.filter(({ question }) => question.is_active === false);

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
    };
  }

  async function checkUserSubmission() {
    if (!allowResultPreview || autoCheckDisabled) return;
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

    const answers: Answer[] = [];
    for (const [key, value] of formData.entries()) {
      if (key.startsWith('question-') && typeof value === 'string') {
        const questionId = parseInt(key.split('-')[1], 10);
        answers.push({ question_id: questionId, answer: value });
      }
    }

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
    } catch (error) {
      const message = (error as Error).message || 'Error al enviar el examen';
      dispatchExamUi({ type: 'SUBMIT_ERROR', payload: message });
      setFormError(message);
    }
  };

  function renderSubmissionSummary(message: string) {
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
      </div>
    );
  }

  function renderQuestionCard(
    entry: { index: number; question: Question },
    position: number,
    isReserve: boolean,
  ) {
    const { question, index: originalIndex } = entry;
    const isCancelled = question.is_cancelled === true;
    const displayName = question.name ?? position;
    const label = `${isReserve ? 'Reserva' : 'Pregunta'} ${displayName}`;
    const key = question.id ?? `${isReserve ? 'reserve' : 'active'}-${position}-${originalIndex}`;
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
        className="bg-[#2a2d33] p-6 mb-4 mt-4 rounded-lg shadow-lg border border-transparent hover:border-purple-500/40 transition-colors"
        key={key}
      >
        <div className="flex items-center justify-between mb-4">
          <p className="font-bold text-lg text-purple-200">{label}</p>
          <span className={`text-xs font-semibold px-2 py-1 rounded ${badgeConfig.className}`}>
            {badgeConfig.text}
          </span>
        </div>
        <ul className="list-none p-0 m-0 flex flex-wrap gap-4">
          {ANSWER_OPTIONS.map((optionChar) => (
            <li key={optionChar} className="mb-0">
              <input
                type="radio"
                name={`question-${question.id}`}
                value={optionChar}
                id={`option-${question.id}-${optionChar}`}
                required={!isCancelled}
                className="mr-1 transform scale-125 cursor-pointer"
              />
              <label
                htmlFor={`option-${question.id}-${optionChar}`}
                className="text-gray-300 text-lg cursor-pointer"
              >
                {optionChar}
              </label>
            </li>
          ))}
        </ul>
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

  return (
    <main>
      <a
        href="/"
        className="inline-block mb-6 px-4 py-2 font-bold text-purple-300 border border-purple-300 rounded-md no-underline transition-colors duration-300 ease-in-out hover:bg-purple-300 hover:text-[#1a1c22]"
      >
        &larr; Volver a la seleccion de examen
      </a>
      <h1 className="text-5xl font-extrabold leading-tight text-center mb-12 text-purple-300 shadow-purple-500/50">{examName}</h1>

      <form id="exam-form" onSubmit={handleSubmit} noValidate>
        <div className="mb-6">
          <label htmlFor="name" className="block font-bold text-purple-500 mb-2">Nombre:</label>
          <input
            type="text"
            id="name"
            name="name"
            required
            value={studentName}
            onInput={(event) => setStudentName((event.target as HTMLInputElement).value)}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-purple-400 focus:ring-2 focus:ring-purple-400/50"
          />
        </div>

        <div className="mb-6">
          <label htmlFor="surname" className="block font-bold text-purple-500 mb-2">Apellidos:</label>
          <input
            type="text"
            id="surname"
            name="surname"
            required
            value={studentSurname}
            onInput={(event) => setStudentSurname((event.target as HTMLInputElement).value)}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-purple-400 focus:ring-2 focus:ring-purple-400/50"
          />
        </div>

        <div className="mb-6">
          <label htmlFor="email" className="block font-bold text-purple-500 mb-2">Email:</label>
          <input
            type="email"
            id="email"
            name="email"
            required
            value={email}
            onInput={(event) => setEmail((event.target as HTMLInputElement).value)}
            onBlur={checkUserSubmission}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-purple-400 focus:ring-2 focus:ring-purple-400/50"
          />
        </div>

        <div className="mb-6">
          <label htmlFor="dni" className="block font-bold text-purple-500 mb-2">DNI/NIE:</label>
          <input
            type="text"
            id="dni"
            name="dni"
            required
            value={dni}
            onInput={(event) => setDni(normalizeDni((event.target as HTMLInputElement).value))}
            onBlur={checkUserSubmission}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-purple-400 focus:ring-2 focus:ring-purple-400/50"
          />
        </div>

        {checking && (
          <p className="text-center text-purple-300 bg-purple-300/10 border border-purple-400 p-4 rounded-md mt-6">
            Comprobando si ya has entregado el examen...
          </p>
        )}
        {submissionMessage && renderSubmissionSummary(submissionMessage)}
        {resultError && (
          <p className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mt-6">
            {resultError}
          </p>
        )}

        {!hasPreviousSubmission && (
          <>
            <section>
              <h2 className="text-2xl font-semibold text-purple-300 mt-8">Preguntas activas</h2>
              <p className="text-sm text-gray-400 mt-1">Responde todas las preguntas activas; puntuan en la nota final.</p>
              {activeEntries.map((entry, index) => renderQuestionCard(entry, index + 1, false))}
            </section>

            {reserveEntries.length > 0 && (
              <section className="mt-8">
                <h2 className="text-2xl font-semibold text-purple-300">Preguntas de reserva</h2>
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
                  className="mt-1 h-4 w-4 rounded border border-[#555] bg-[#2a2d33] text-purple-500 focus:ring-2 focus:ring-purple-400/60"
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
                  className="text-purple-300 underline decoration-dotted hover:text-purple-200"
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
            className="w-full py-3 text-lg cursor-pointer font-bold mt-4 rounded-md bg-purple-600 hover:bg-purple-700 transition-colors disabled:opacity-70 disabled:cursor-not-allowed"
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
