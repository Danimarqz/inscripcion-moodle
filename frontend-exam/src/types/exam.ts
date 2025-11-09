export type Exam = {
  id: number;
  name: string;
  score?: number | null;
  percentile?: number | null;
  show_score?: boolean;
  show_percentile?: boolean;
  show_score_full?: boolean;
};
export type ExamOut = {
  score?: number | null;
  percentile?: number | null;
  position?: number | null;
  total_submissions?: number | null;
  correct_answers?: number | null;
  total_questions?: number | null;
  message?: string | null;
};

export type Question = {
  id: number;
  correct_option?: string;
  is_active: boolean;
}

export type Answer = {
  question_id: number;
  answer: string;
}

export type SubmissionAnswer = {
  id: number;
  question_id: number;
  answer: string;
}

export type ExamUser = {
  id: number;
  name: string;
  surname: string;
  email?: string | null;
  dni: string;
};

export type AdminSubmission = {
  id: number;
  exam_id: number;
  user?: ExamUser | null;
  email?: string | null;
  dni: string;
  name: string;
  surname: string;
  score?: number | null;
  percentile?: number | null;
  submitted_at: string;
  answers: SubmissionAnswer[];
};

export type ExamSubmissionPayload = {
  email: string;
  dni: string;
  name: string;
  surname: string;
  exam_id: number;
  answers: Answer[];
}

export type SubmissionUpdatePayload = {
  email: string;
  dni: string;
  name: string;
  surname: string;
  answers: Answer[];
};

export type ExamQuestionsResponse = {
  exam_name: string;
  questions: Question[];
};

export type QuestionCreate = {
  id?: number;
  correct_option: string;
  is_active?: boolean;
}

export type ExamCreateWithQuestions = {
  name: string;
  is_active?: boolean;
  show_score?: boolean;
  show_percentile?: boolean;
  show_score_full?: boolean;
  questions: QuestionCreate[];
};

export type QuestionEdit = {
  id?: number;
  correct_option: string;
  is_active?: boolean;
}

export type ExamEdit = {
  id?: number;
  name?: string;
  is_active?: boolean;
  show_score?: boolean;
  show_percentile?: boolean;
  show_score_full?: boolean;
  questions: QuestionEdit[];
};

export type UserSubmissionCheck = {
  email: string;
  dni: string;
  exam_id: number;
};

export type ExamOfficialResult = {
  id: number;
  exam_id: number;
  user?: ExamUser | null;
  dni_masked: string;
  apellido_1: string;
  apellido_2?: string | null;
  nombre: string;
  created_at: string;
};

export type ImportOfficialResultsSummary = {
  exam_id: number;
  total_rows: number;
  imported_results: number;
  created_users: number;
  updated_users: number;
};

export type SortableHeaderProps = {
  label: string;
  sortKey: 'dni' | 'nombre' | 'apellidos' | 'usuario' | 'creado';
  activeKey: 'dni' | 'nombre' | 'apellidos' | 'usuario' | 'creado';
  direction: 'asc' | 'desc';
  onSort: (key: 'dni' | 'nombre' | 'apellidos' | 'usuario' | 'creado') => void;
};

export type ExamUiState = {
  checking: boolean;
  hasPreviousSubmission: boolean;
  score: number | null;
  percentile: number | null;
  position: number | null;
  totalSubmissions: number | null;
  correctAnswers: number | null;
  totalQuestions: number | null;
  submissionMessage: string | null;
  resultError: string | null;
};

export type ExamResultPayload = {
  score: number | null;
  percentile: number | null;
  position: number | null;
  totalSubmissions: number | null;
  correctAnswers: number | null;
  totalQuestions: number | null;
  message: string;
};

export type ExamUiAction =
  | { type: 'RESET' }
  | { type: 'CHECK_START' }
  | { type: 'CHECK_SUCCESS'; payload: ExamResultPayload }
  | { type: 'CHECK_ERROR'; payload?: string | null }
  | { type: 'SUBMIT_SUCCESS'; payload: ExamResultPayload }
  | { type: 'SUBMIT_ERROR'; payload?: string | null };


