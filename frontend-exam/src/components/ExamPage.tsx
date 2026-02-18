import type { JSX } from 'preact';
import { useCallback, useEffect, useMemo, useState } from 'preact/hooks';

import type {
  Answer,
  AnswerReview,
  ExamOut,
  ExamResultPayload,
  ExamSubmissionPayload,
} from '../types/exam';
import { checkOfficialResult, checkSubmission, submitExam } from '../services/examService';
import { useExamQuestions } from '../hooks/useExamQuestions';
import { useExamUi } from '../hooks/useExamUi';
import { isValidEmail, normalizeDni, validateDniNie } from '../utils/validation';
import { buildResultPayload } from '../utils/examLogic';
import QuestionList from './questions/QuestionList';
import QuickResultCheck from './submissions/QuickResultCheck';
import SubmissionSummary from './submissions/SubmissionSummary';
import EligibilityModal from './modals/EligibilityModal';
import SubmissionIdentityFields from './submissions/SubmissionIdentityFields';


interface ExamPageProps {
  examId: number;
  examName: string;
  showScore: boolean;
  showPercentile: boolean;
  showScoreFull: boolean;
  validatedTribunal?: boolean;
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
  const [resultType, setResultType] = useState('General');
  const [acceptsMarketing, setAcceptsMarketing] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [autoCheckDisabled, setAutoCheckDisabled] = useState(false);
  const [userAnswers, setUserAnswers] = useState<Record<number, string>>({});
  const [eligibilityAllowed, setEligibilityAllowed] = useState(false);
  const [eligibilityChecking, setEligibilityChecking] = useState(false);
  const [eligibilityError, setEligibilityError] = useState<string | null>(null);
  const [showEligibilityModal, setShowEligibilityModal] = useState(false);

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
    maxScore,
    secondaryMaxScores,
  } = examUiState;

  const { questions, loading, error: questionsError } = useExamQuestions(examId);
  const allowResultPreview =
    showScore || showPercentile || showScoreFull || validatedTribunal;

  useEffect(() => {
    dispatchExamUi({ type: 'RESET' });
    setAutoCheckDisabled(false);
    setAcceptsMarketing(false);
    setUserAnswers({});
    setEligibilityAllowed(false);
    setEligibilityError(null);
    setShowEligibilityModal(false);
  }, [examId, dispatchExamUi]);

  useEffect(() => {
    setEligibilityAllowed(false);
    setEligibilityError(null);
    setShowEligibilityModal(false);

    const trimmedName = studentName.trim();
    const trimmedSurname = studentSurname.trim();
    const normalizedDni = normalizeDni(dni);

    if (!trimmedName || !trimmedSurname) {
      setEligibilityChecking(false);
      return;
    }
    if (!validateDniNie(normalizedDni)) {
      setEligibilityChecking(false);
      return;
    }

    let cancelled = false;
    setEligibilityChecking(true);
    const timer = window.setTimeout(() => {
      void checkOfficialResult({
        exam_id: examId,
        name: trimmedName,
        surname: trimmedSurname,
        dni: normalizedDni,
      })
        .then((res) => {
          if (cancelled) return;
          setEligibilityAllowed(res.match);
          if (!res.match) {
            setShowEligibilityModal(true);
          }
        })
        .catch((err) => {
          if (cancelled) return;
          setEligibilityError(err instanceof Error ? err.message : 'No se pudo comprobar el acceso');
        })
        .finally(() => {
          if (cancelled) return;
          setEligibilityChecking(false);
        });
    }, 350);

    return () => {
      cancelled = true;
      window.clearTimeout(timer);
      setEligibilityChecking(false);
    };
  }, [studentName, studentSurname, dni, examId]);

  const questionEntries = useMemo(
    () => questions.map((question, index) => ({ index, question })),
    [questions],
  );
  const activeEntries = questionEntries.filter(({ question }) => question.is_active !== false);
  const reserveEntries = questionEntries.filter(({ question }) => question.is_active === false);

  const setAnswerForQuestion = useCallback((questionId: number, option: string) => {
    const normalizedOption = option.toUpperCase();
    setUserAnswers((prev) => ({
      ...prev,
      [questionId]: normalizedOption,
    }));
  }, []);

  const clearAnswerForQuestion = useCallback((questionId: number) => {
    setUserAnswers((prev) => {
      if (!(questionId in prev)) {
        return prev;
      }
      const next = { ...prev };
      delete next[questionId];
      return next;
    });
  }, []);



  function getResultPayload(result: ExamOut, context: 'check' | 'submit'): ExamResultPayload {
    return buildResultPayload(result, context, {
      showScore,
      showPercentile,
      showScoreFull,
    });
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
      const payload = getResultPayload(data, 'check');
      dispatchExamUi({ type: 'CHECK_SUCCESS', payload });
      setAutoCheckDisabled(true);
    } catch (error) {
      const message = (error as Error).message;
      if (message && message !== 'submission not found') {
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
    if (!eligibilityAllowed) {
      setFormError('Debemos verificar que participaste en el examen oficial antes de continuar.');
      setShowEligibilityModal(true);
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

    const meritsRaw = formData.get('merits') as string;
    const merits = meritsRaw ? parseFloat(meritsRaw) : undefined;

    const payload: ExamSubmissionPayload = {
      email: emailRaw,
      dni: dniRaw,
      name: trimmedName,
      surname: trimmedSurname,
      exam_id: examId,
      answers,
      merits,
      accepts_marketing: acceptsMarketing,
      eligibility_confirmed: eligibilityAllowed,
      result_type: resultType,
    };

    try {
      const result = await submitExam(payload);
      const payloadResult = getResultPayload(result, 'submit');

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
      const lower = message.toLowerCase();
      const isEligibilityIssue =
        lower.includes('examen oficial') ||
        lower.includes('verificado') ||
        lower.includes('resultado') ||
        lower.includes('registrar tus resultados');

      if (isEligibilityIssue) {
        setShowEligibilityModal(true);
        setEligibilityAllowed(false);
        setFormError(null);
      } else {
        setFormError(message);
      }

      dispatchExamUi({ type: 'SUBMIT_ERROR', payload: message });
    }
  };



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

  const resultsSummary =
    submissionMessage ? (
      <SubmissionSummary
        message={submissionMessage}
        review={answersReview}
        showScore={showScore}
        showScoreFull={showScoreFull}
        showPercentile={showPercentile}
        score={latestScore}
        correctAnswers={correctAnswers}
        totalQuestions={totalQuestions}
        percentile={latestPercentile}
        position={position}
        totalSubmissions={totalSubmissions}
        maxScore={maxScore}
        secondaryMaxScores={secondaryMaxScores}
      />
    ) : null;

  return (
    <main>
      <EligibilityModal open={showEligibilityModal} onClose={() => setShowEligibilityModal(false)} />
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
        <QuickResultCheck
          email={email}
          dni={dni}
          checking={checking}
          onEmailChange={setEmail}
          onDniChange={setDni}
          onSubmit={handleManualCheck}
        />
      )}
      {allowResultPreview && resultsSummary}
      {validatedTribunal && (
                <p className="text-xs text-brand-yellow mt-2">
                  Estas respuestas NO las puedes modificar, si te has confundido al meter alguna respuesta envíanos un correo a <a href="mailto:info.opositatcae@gmail.com">info.opositatcae@gmail.com</a> y lo corregiremos.
                </p>
              )}
      <form id="exam-form" onSubmit={handleSubmit} noValidate>
        <SubmissionIdentityFields
          studentName={studentName}
          studentSurname={studentSurname}
          email={email}
          dni={dni}
          onNameChange={setStudentName}
          onSurnameChange={setStudentSurname}
          onEmailChange={setEmail}
          onDniChange={setDni}
          onBlurCheck={handleCheckSubmissionBlur}
          eligibilityError={eligibilityError}
        />
        
        {!hasPreviousSubmission && (
          <div className="mt-6 mb-6">
            <label className="block text-sm font-medium text-gray-300 mb-2">
              Tipo de convocatoria
            </label>
            <select
              value={resultType}
              onChange={(e) => setResultType((e.target as HTMLSelectElement).value)}
              className="w-full px-4 py-3 rounded-lg bg-[#2a2d33] border border-[#555] text-white focus:ring-2 focus:ring-brand-pink focus:border-transparent transition-all"
            >
              <option value="General">General</option>
              <option value="Promoción interna">Promoción interna</option>
              <option value="Discapacidad">Discapacidad</option>
              <option value="Otros">Otros</option>
            </select>
          </div>
        )}

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
            <QuestionList
              activeEntries={activeEntries}
              reserveEntries={reserveEntries}
              userAnswers={userAnswers}
              onSetAnswer={setAnswerForQuestion}
              onClearAnswer={clearAnswerForQuestion}
            />

            <div className="mt-8 mb-6 p-4 rounded-lg bg-[#2a2d33] border border-[#555]">
               <label className="block text-sm font-medium text-gray-300 mb-2">
                  Puntuación de méritos (Opcional)
               </label>
               <input
                 type="number"
                 step="0.01"
                 min="0"
                 placeholder="0"
                 name="merits"
                 className="w-full px-4 py-3 rounded-lg bg-[#1f2229] border border-[#555] text-white focus:ring-2 focus:ring-brand-pink focus:border-transparent transition-all"
               />
            </div>

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
            className="btn-brand w-full text-lg mt-4 disabled:opacity-60 disabled:cursor-not-allowed cursor-pointer"
            disabled={!eligibilityAllowed || eligibilityChecking}
          >
            {eligibilityChecking ? 'Comprobando...' : 'Entregar Examen'}
          </button>
        )}
        {formError && (
          <p className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mt-6">{formError}</p>
        )}
      </form>
    </main>
  );
}
