export interface Exam {
  id: number;
  name: string;
  is_active: boolean;
  show_responses: boolean;
}

export interface Question {
  id: number;
}

export interface Answer {
  question_id: number;
  answer: string;
}

export interface ExamSubmissionPayload {
  email: string;
  dni: string;
  exam_id: number;
  answers: Answer[];
}

export interface ExamQuestionsResponse {
  exam_name: string;
  questions: Question[];
}