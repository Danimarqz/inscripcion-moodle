export type Exam = {
  id: number;
  name: string;
  is_active: boolean;
  show_response: boolean;
}

export type Question = {
  id: number;
  correct_option?: string;
}

export type Answer = {
  question_id: number;
  answer: string;
}

export type ExamSubmissionPayload = {
  email: string;
  dni: string;
  exam_id: number;
  answers: Answer[];
}

export type ExamQuestionsResponse = {
  exam_name: string;
  questions: Question[];
}

export type QuestionCreate = {
  id?: number;
  correct_option: string;
}

export type ExamCreateWithQuestions = {
  name: string;
  is_active?: boolean;
  show_response?: boolean;
  questions: QuestionCreate[];
}

export type QuestionEdit = {
  id?: number;
  correct_option: string;
}

export type ExamEdit = {
  name?: string;
  is_active?: boolean;
  show_response?: boolean;
  questions: QuestionEdit[];
}
