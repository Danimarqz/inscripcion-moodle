import { useEffect, useState } from 'preact/hooks';

import type { Answer, Exam, ExamSubmissionPayload, Question } from '../types/exam';
import { getQuestions, submitExam, checkSubmission } from '../services/examService';
import { normalizeDni, validateDniNie } from '../utils/validation';

interface ExamPageProps {
  examId: number;
  examName: string;
  showResponse: boolean;
}

const ANSWER_OPTIONS = ['A', 'B', 'C', 'D'];

function isValidEmail(value: string) {
  return /^[\w-.]+@([\w-]+\.)+[\w-]{2,}$/i.test(value.trim());
}

export default function ExamPage({ examId, examName, showResponse }: ExamPageProps) {
  const [questions, setQuestions] = useState<Question[]>([]);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const [studentName, setStudentName] = useState('');
  const [studentSurname, setStudentSurname] = useState('');
  const [email, setEmail] = useState('');
  const [dni, setDni] = useState('');
  const [score, setScore] = useState<number | null>(null);
  const [percentile, setPercentile] = useState<number | null>(null);
  const [checkingResult, setCheckingResult] = useState(false);
  const [resultError, setResultError] = useState<string | null>(null);
  const [submissionMessage, setSubmissionMessage] = useState<string | null>(null);

  useEffect(() => {
    async function fetchQuestions() {
      try {
        const data = await getQuestions(examId);
        setQuestions(data);
      } catch (error) {
        setErrorMessage((error as Error).message);
      } finally {
        setLoading(false);
      }
    }

    fetchQuestions();
  }, [examId]);

  async function checkUserSubmission() {
    if (!showResponse) return;
    setResultError(null);
    setScore(null);
    setPercentile(null);
    setSubmissionMessage(null);

    const normalizedEmail = email.trim().toLowerCase();
    const normalizedDni = normalizeDni(dni);

    if (!isValidEmail(normalizedEmail) || !validateDniNie(normalizedDni)) return;
    setCheckingResult(true);
    try {
      const data: Exam = await checkSubmission({ email: normalizedEmail, dni: normalizedDni, exam_id: examId });
      setScore(data.score ?? 0);
      setPercentile(data.percentile ?? 0);
      const message = `Ya has entregado el examen. Tu puntuacion es: ${data.score ?? 'Procesando'}. Percentil: ${
        data.percentile ?? 'N/A'
      }`;
      setSubmissionMessage(message);
    } catch (e) {
      setResultError((e as Error).message);
    } finally {
      setCheckingResult(false);
    }
  }

  const handleSubmit = async (event: Event) => {
    event.preventDefault();
    setErrorMessage(null);

    const trimmedName = studentName.trim();
    const trimmedSurname = studentSurname.trim();

    if (!trimmedName) {
      setErrorMessage('El nombre es obligatorio.');
      return;
    }
    if (!trimmedSurname) {
      setErrorMessage('Los apellidos son obligatorios.');
      return;
    }

    const form = event.currentTarget as HTMLFormElement;
    const formData = new FormData(form);

    const emailRaw = (formData.get('email') as string).trim().toLowerCase();
    const dniRaw = normalizeDni(formData.get('dni') as string);

    if (!isValidEmail(emailRaw)) {
      setErrorMessage('Por favor, introduce un email valido.');
      return;
    }
    if (!validateDniNie(dniRaw)) {
      setErrorMessage('Por favor, introduce un DNI o NIE valido.');
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

      if (showResponse) {
        setScore(result.score ?? null);
        setPercentile(result.percentile ?? null);
      }

      const message = `Examen entregado. ${result.message}. Tu puntuacion es: ${
        result.score ?? 'Procesando'
      }. Percentil: ${result.percentile ?? 'N/A'}`;
      setSubmissionMessage(message);
    } catch (error) {
      setErrorMessage((error as Error).message);
    }
  };

  if (loading) return <main>Cargando preguntas...</main>;

  if (questions.length === 0)
    return (
      <main>
        <p className="info-message">No hay preguntas disponibles para este examen.</p>
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
        {submissionMessage && (
          <p className="text-center text-green-400 bg-green-400/10 border border-green-500 p-4 rounded-md mt-6">
            {submissionMessage}
          </p>
        )}
        {!submissionMessage && questions.map((question, index) => (
          <div className="bg-[#2a2d33] p-6 mb-4 mt-4 rounded-lg shadow-lg" key={question.id}>
            <p className="font-bold text-lg mb-4">Pregunta {index + 1}</p>
            <ul className="list-none p-0 m-0 flex flex-wrap gap-4">
              {ANSWER_OPTIONS.map((optionChar) => (
                <li key={optionChar} className="mb-0">
                  <input
                    type="radio"
                    name={`question-${question.id}`}
                    value={optionChar}
                    id={`option-${question.id}-${optionChar}`}
                    required
                    className="mr-2 transform scale-125"
                  />
                  <label htmlFor={`option-${question.id}-${optionChar}`} className="text-gray-300 text-lg cursor-pointer">{optionChar}</label>
                </li>
              ))}
            </ul>
          </div>
        ))}

        {!submissionMessage && (<button
          type="submit"
          className="w-full py-3 text-lg font-bold mt-8 rounded-md bg-purple-600 hover:bg-purple-700 transition-colors disabled:opacity-70 disabled:cursor-not-allowed"
        >
          Entregar Examen
        </button>)}
        {errorMessage && (
          <p className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mt-6">{errorMessage}</p>
        )}
      </form>
    </main>
  );
}
