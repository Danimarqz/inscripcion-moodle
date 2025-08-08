import { useEffect, useState } from 'preact/hooks';
import type { Answer, Question, ExamSubmissionPayload, Exam } from '../types/exam';
import { getQuestions, submitExam, checkSubmission } from '../services/examService';

interface ExamPageProps {
  examId: number;
  examName: string;
  showResponse: boolean;
}

export default function ExamPage({ examId, examName, showResponse }: ExamPageProps) {
  const [questions, setQuestions] = useState<Question[]>([]);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // Estados para validación resultados
  const [email, setEmail] = useState('');
  const [dni, setDni] = useState('');
  const [score, setScore] = useState<number | null>(null);
  const [percentile, setPercentile] = useState<number | null>(null);
  const [checkingResult, setCheckingResult] = useState(false);
  const [resultError, setResultError] = useState<string | null>(null);

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

  function isValidEmail(e: string) {
    const emailRegex = /^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$/;
    return emailRegex.test(e);
  }

  function isValidDni(d: string) {
    const dniRegex = /^[XYZ]?\d{7,8}[A-Z]$/i;
    return dniRegex.test(d);
  }

  async function checkUserSubmission() {
    setResultError(null);
    setScore(null);
    setPercentile(null);

    console.log('checkUserSubmission called', { showResponse, email, dni });

    if (!isValidEmail(email) || !isValidDni(dni)) return;

    setCheckingResult(true);
    try {
      const data: Exam = await checkSubmission({ email, dni, exam_id: examId });
      setScore(data.score || 0);
      setPercentile(data.percentile || 0);
    } catch (e) {
      setResultError((e as Error).message);
    } finally {
      setCheckingResult(false);
    }
  }

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setErrorMessage(null);

    const form = e.currentTarget as HTMLFormElement;
    const formData = new FormData(form);

    const emailRaw = formData.get('email') as string;
    const dniRaw = formData.get('dni') as string;
    const dniVal = dniRaw.toUpperCase();

    if (!isValidEmail(emailRaw)) {
      setErrorMessage('Por favor, introduce un email válido.');
      return;
    }
    if (!isValidDni(dniVal)) {
      setErrorMessage('Por favor, introduce un DNI válido o NIE válido.');
      return;
    }

    const answers: Answer[] = [];
    for (const [key, value] of formData.entries()) {
      if (key.startsWith('question-')) {
        const questionId = parseInt(key.split('-')[1], 10);
        if (typeof value === 'string') {
          answers.push({ question_id: questionId, answer: value });
        }
      }
    }

    const payload: ExamSubmissionPayload = {
      email: emailRaw,
      dni: dniVal,
      exam_id: examId,
      answers,
    };

    try {
      const result = await submitExam(payload);

      alert(`Examen entregado. ${result.message}. Tu puntuación es: ${result.score || 'Procesando'}. Percentil: ${result.percentile || 'N/A'}`);
      window.location.href = '/';
    } catch (error) {
      setErrorMessage((error as Error).message);
    }
  };

  const options = ['A', 'B', 'C', 'D'];

  if (loading) return <main>Cargando preguntas...</main>;

  

  if (questions.length === 0)
    return (
      <main>
        <p className="info-message">No hay preguntas disponibles para este examen.</p>
      </main>
    );

  return (
    <main>
      <a href="/" className="inline-block mb-6 px-4 py-2 font-bold text-purple-300 border border-purple-300 rounded-md no-underline transition-colors duration-300 ease-in-out hover:bg-purple-300 hover:text-[#1a1c22]">&larr; Volver a la selección de examen</a>
      <h1 className="text-5xl font-extrabold leading-tight text-center mb-12 text-purple-300 shadow-purple-500/50">{examName}</h1>

      <form id="exam-form" onSubmit={handleSubmit} noValidate>
        <div className="mb-6">
          <label htmlFor="email" className="block font-bold text-purple-500 mb-2">Email:</label>
          <input
            type="email"
            id="email"
            name="email"
            required
            value={email}
            onInput={e => setEmail((e.target as HTMLInputElement).value)}
            onBlur={checkUserSubmission}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-purple-400 focus:ring-2 focus:ring-purple-400/50"
          />
        </div>

        <div className="mb-6">
          <label htmlFor="dni" className="block font-bold text-purple-500 mb-2">DNI:</label>
          <input
            type="text"
            id="dni"
            name="dni"
            required
            pattern="^[XYZ]?\d{7,8}[A-Z]$"
            title="Introduce un DNI (8 números y 1 letra) o NIE (X, Y o Z seguido de 7 u 8 números y 1 letra) válido."
            value={dni}
            onInput={e => setDni((e.target as HTMLInputElement).value.toUpperCase())}
            onBlur={checkUserSubmission}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-purple-400 focus:ring-2 focus:ring-purple-400/50"
          />
        </div>

        {questions.map((question, index) => (
          <div className="bg-[#2a2d33] p-6 mb-4 rounded-lg shadow-lg" key={question.id}>
            <p className="font-bold text-lg mb-4">Pregunta {index + 1}</p>
            <ul className="list-none p-0 m-0 flex flex-wrap gap-4">
              {options.map((optionChar) => (
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

        <button type="submit" className="w-full py-3 text-lg font-bold mt-8 rounded-md bg-purple-600 hover:bg-purple-700 transition-colors disabled:opacity-70 disabled:cursor-not-allowed">
          Entregar Examen
        </button>
        {errorMessage && (
          <p className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mt-6">{errorMessage}</p>
        )}
      </form>

      {showResponse && (
        <section className="mt-12">
          <h2 className="text-3xl font-bold mb-6">Resultados</h2>
          {checkingResult && <p>Comprobando resultados...</p>}
          {!checkingResult && score !== null && percentile !== null && (
            <p className="text-xl">
              Tu puntuación: <strong>{score}</strong> <br />
              Percentil: <strong>{percentile}</strong>
            </p>
          )}
          {!checkingResult && resultError && <p className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md">{resultError}</p>}
        </section>
      )}
    </main>
  );
}
