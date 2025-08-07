import { useEffect, useState } from 'preact/hooks';
import type { Answer, Question, ExamSubmissionPayload } from '../types/exam';
import { getQuestions, submitExam } from '../services/examService';

interface ExamPageProps {
  examId: number;
  examName: string;
}

export default function ExamPage({ examId, examName }: ExamPageProps) {
  const [questions, setQuestions] = useState<Question[]>([]);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

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

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setErrorMessage(null);

    const form = e.currentTarget as HTMLFormElement;
    const formData = new FormData(form);

    const email = formData.get('email') as string;
    const dniRaw = formData.get('dni') as string;
    const dni = dniRaw.toUpperCase();

    // Validaciones
    const emailRegex = /^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$/;
    if (!emailRegex.test(email)) {
      setErrorMessage('Por favor, introduce un email válido.');
      return;
    }
    const dniRegex = /^[XYZ]?\d{7,8}[A-Z]$/i;
    if (!dniRegex.test(dni)) {
      setErrorMessage('Por favor, introduce un DNI válido o NIE válido.');
      return;
    }

    // Recoger respuestas
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
      email,
      dni,
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
      {errorMessage && <p className="error-message">{errorMessage}</p>}

      <form id="exam-form" onSubmit={handleSubmit} noValidate>
        <div className="form-group">
          <label htmlFor="email">Email:</label>
          <input type="email" id="email" name="email" required />
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
    </main>
  );
}
