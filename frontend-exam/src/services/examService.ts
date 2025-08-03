import type { Exam, Question, Answer, ExamSubmissionPayload, ExamQuestionsResponse } from '../types/exam';

const API_URL = import.meta.env.PUBLIC_API_URL;

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
  return await response.json() as Question[];
}

export async function submitExam(payload: ExamSubmissionPayload): Promise<{ score: number, percentile: number, message: string }> {
  console.log(JSON.stringify(payload))
  const response = await fetch(`${API_URL}/submit-exam`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    throw new Error('Error submitting exam');
  }
  return await response.json();
}
