import type {
  Answer,
  Exam,
  ExamOut,
  ExamQuestionsResponse,
  ExamSubmissionPayload,
  Question,
  UserSubmissionCheck,
} from '../types/exam';

const API_URL = "https://simulador.opositatcae.es/api";

export async function getExams(): Promise<Exam[]> {
  const response = await fetch(`${API_URL}/exams`);
  if (!response.ok) {
    throw new Error('Error fetching exams');
  }
  return await response.json();
}

export async function getQuestions(examId: number): Promise<Question[]> {
  const response = await fetch(`${API_URL}/exams/${examId}/questions`);
  if (!response.ok) {
    throw new Error('Error fetching questions');
  }
  return (await response.json()) as Question[];
}

export async function submitExam(payload: ExamSubmissionPayload): Promise<ExamOut> {
  const response = await fetch(`${API_URL}/submit-exam`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.detail || 'Error submitting exam');
  }
  return data;
}

export async function checkSubmission(payload: UserSubmissionCheck): Promise<ExamOut> {
  const response = await fetch(`${API_URL}/check_submission`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  const data = await response.json();
  if (!response.ok) {
    const message = typeof data?.detail === 'string' ? data.detail : 'Error al consultar resultados';
    throw new Error(message);
  }

  return data;
}
