import { useEffect, useMemo, useState } from 'preact/hooks';

import type { Answer, Exam, ExamSubmissionPayload, Question } from '../types/exam';
import { checkSubmission, submitExam } from '../services/examService';
import { useExamQuestions } from '../hooks/useExamQuestions';
import { useExamUi } from '../hooks/useExamUi';
import { isValidEmail, normalizeDni, roundToTwoDecimals, validateDniNie } from '../utils/validation';

interface ExamPageProps {
  examId: number;
  examName: string;
  showResponse: boolean;
}

const ANSWER_OPTIONS = ['A', 'B', 'C', 'D'];

export default function ExamPage({ examId, examName, showResponse }: ExamPageProps) {
  const [studentName, setStudentName] = useState('');
  const [studentSurname, setStudentSurname] = useState('');
  const [email, setEmail] = useState('');
  const [dni, setDni] = useState('');
  const [formError, setFormError] = useState<string | null>(null);

  const [examUiState, dispatchExamUi] = useExamUi();
  const { checking, hasPreviousSubmission, submissionMessage, resultError } = examUiState;

  const { questions, loading, error: questionsError } = useExamQuestions(examId);

  useEffect(() => {
    dispatchExamUi({ type: 'RESET' });
  }, [examId]);

  const questionEntries = useMemo(
    () => questions.map((question, index) => ({ index, question })),
    [questions],
  );
  const activeEntries = questionEntries.filter(({ question }) => question.is_active !== false);
  const reserveEntries = questionEntries.filter(({ question }) => question.is_active === false);

  async function checkUserSubmission() {
    if (!showResponse) return;
    if (checking) return;

    const normalizedEmail = email.trim().toLowerCase();
    const normalizedDni = normalizeDni(dni);

    if (!isValidEmail(normalizedEmail) || !validateDniNie(normalizedDni)) return;

    dispatchExamUi({ type: 'CHECK_START' });

    try {
      const data: Exam = await checkSubmission({ email: normalizedEmail, dni: normalizedDni, exam_id: examId });
      const nextScore = roundToTwoDecimals(data.score);
      const nextPercentile = roundToTwoDecimals(data.percentile);
      const message = `Ya has entregado el examen. Tu puntuacion es: ${nextScore ?? 'N/A'}. Percentil: ${
        nextPercentile ?? 'N/A'
      }`;
      dispatchExamUi({
        type: 'CHECK_SUCCESS',
        payload: { score: nextScore, percentile: nextPercentile, message },
      });
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
    };

    try {
      const result = await submitExam(payload);

      let nextScore: number | null = null;
      let nextPercentile: number | null = null;

      if (showResponse) {
        nextScore = roundToTwoDecimals(result.score);
        nextPercentile = roundToTwoDecimals(result.percentile);
      }

      const baseMessage = result.message ?? 'Tu entrega ha sido registrada correctamente';
      const message = `Examen entregado. ${baseMessage}. Tu puntuacion es: ${
        nextScore ?? 'N/A'
      }. Percentil: ${nextPercentile ?? 'N/A'}`;

      dispatchExamUi({
        type: 'SUBMIT_SUCCESS',
        payload: { score: nextScore, percentile: nextPercentile, message },
      });
      setStudentName('');
      setStudentSurname('');
      setEmail('');
      setDni('');
    } catch (error) {
      const message = (error as Error).message || 'Error al enviar el examen';
      dispatchExamUi({ type: 'SUBMIT_ERROR', payload: message });
      setFormError(message);
    }
  };

  function renderQuestionCard(
    entry: { index: number; question: Question },
    position: number,
    isReserve: boolean,
  ) {
    const { question, index: originalIndex } = entry;
    const label = `Pregunta ${position}`;
    const badgeClass = isReserve
      ? 'bg-amber-500/20 text-amber-300 border border-amber-500/50'
      : 'bg-green-500/20 text-green-300 border border-green-500/50';
    const key = question.id ?? `${isReserve ? 'reserve' : 'active'}-${position}-${originalIndex}`;

    return (
      <div
        className="bg-[#2a2d33] p-6 mb-4 mt-4 rounded-lg shadow-lg border border-transparent hover:border-purple-500/40 transition-colors"
        key={key}
      >
        <div className="flex items-center justify-between mb-4">
          <p className="font-bold text-lg text-purple-200">{label}</p>
          {isReserve && <span className={`text-xs font-semibold px-2 py-1 rounded ${badgeClass}`}>{
            isReserve ? 'Reserva' : 'Activa'
          }</span>}
        </div>
        <ul className="list-none p-0 m-0 flex flex-wrap gap-4">
          {ANSWER_OPTIONS.map((optionChar) => (
            <li key={optionChar} className="mb-0">
              <input
                type="radio"
                name={`question-${question.id}`}
                value={optionChar}
                id={`option-${question.id}-${optionChar}`}
                required
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
        {submissionMessage && (
          <p className="text-center text-green-400 bg-green-400/10 border border-green-500 p-4 rounded-md mt-6">
            {submissionMessage}
          </p>
        )}
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
          </>
        )}

        {!hasPreviousSubmission && (
          <button
            type="submit"
            className="w-full py-3 text-lg cursor-pointer font-bold mt-8 rounded-md bg-purple-600 hover:bg-purple-700 transition-colors disabled:opacity-70 disabled:cursor-not-allowed"
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
