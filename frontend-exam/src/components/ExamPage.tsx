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

    if (!showResponse || !isValidEmail(email) || !isValidDni(dni)) return;

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

  if (errorMessage)
    return (
      <main>
        <p className="error-message">{errorMessage}</p>
      </main>
    );

  if (questions.length === 0)
    return (
      <main>
        <p className="info-message">No hay preguntas disponibles para este examen.</p>
      </main>
    );

  return (
    <main>
      <a href="/" className="back-button">&larr; Volver a la selección de examen</a>
      <h1>{examName}</h1>

      <form id="exam-form" onSubmit={handleSubmit} noValidate>
        <div className="form-group">
          <label htmlFor="email">Email:</label>
          <input
            type="email"
            id="email"
            name="email"
            required
            value={email}
            onInput={e => setEmail((e.target as HTMLInputElement).value)}
            onBlur={checkUserSubmission}
          />
        </div>

        <div className="form-group">
          <label htmlFor="dni">DNI:</label>
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
          />
        </div>

        {questions.map((question, index) => (
          <div className="question-block" key={question.id}>
            <p className="question-text">Pregunta {index + 1}</p>
            <ul className="options-list">
              {options.map((optionChar) => (
                <li key={optionChar}>
                  <input
                    type="radio"
                    name={`question-${question.id}`}
                    value={optionChar}
                    id={`option-${question.id}-${optionChar}`}
                    required
                  />
                  <label htmlFor={`option-${question.id}-${optionChar}`}>{optionChar}</label>
                </li>
              ))}
            </ul>
          </div>
        ))}

        <button type="submit" className="submit-button">
          Entregar Examen
        </button>
      </form>

      {showResponse && (
        <section className="results-section">
          <h2>Resultados</h2>
          {checkingResult && <p>Comprobando resultados...</p>}
          {!checkingResult && score !== null && percentile !== null && (
            <p>
              Tu puntuación: <strong>{score}</strong> <br />
              Percentil: <strong>{percentile}</strong>
            </p>
          )}
          {!checkingResult && resultError && <p className="error-message">{resultError}</p>}
        </section>
      )}
    </main>
  );
}
