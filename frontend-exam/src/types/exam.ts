export type Exam = {
  id: number;
  name: string;
  score?: number | null;
  percentile?: number | null;
  show_score?: boolean;
  show_percentile?: boolean;
  show_score_full?: boolean;
  validated_tribunal?: boolean;
};
export type ExamOut = {
  score?: number | null;
  percentile?: number | null;
  position?: number | null;
  total_submissions?: number | null;
  correct_answers?: number | null;
  total_questions?: number | null;
  message?: string | null;
  answers_review?: AnswerReview[] | null;
};

export type Question = {
  id: number;
  name: number;
  correct_option?: string;
  is_active: boolean;
  is_cancelled?: boolean;
};

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
  moodle_id?: number | null;
  accepts_marketing?: boolean;
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
  accepts_marketing?: boolean | null;
  selected_result_type?: string | null;
};

export type AdminSubmissionsResponse = {
  submissions: AdminSubmission[];
  total_submissions: number;
  average_score: number | null;
  stats_included: boolean;
};

export type SyncMoodleUsersResponse = {
  checked: number;
  synced: number;
  failed: number;
};

export type ExamSubmissionPayload = {
  email: string;
  dni: string;
  name: string;
  surname: string;
  exam_id: number;
  answers: Answer[];
  accepts_marketing: boolean;
  eligibility_confirmed?: boolean;
  result_type: string;
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

export type OfficialResultCheckPayload = {
  exam_id: number;
  name: string;
  surname: string;
  dni: string;
};

export type OfficialResultCheckResponse = {
  match: boolean;
};

export type QuestionCreate = {
  id?: number;
  name?: number;
  correct_option: string;
  is_active?: boolean;
  is_cancelled?: boolean;
};

export type ExamCreateWithQuestions = {
  name: string;
  is_active?: boolean;
  show_score?: boolean;
  show_percentile?: boolean;
  show_score_full?: boolean;
  validated_tribunal?: boolean;
  questions: QuestionCreate[];
};

export type QuestionEdit = {
  id?: number;
  name?: number;
  correct_option: string;
  is_active?: boolean;
  is_cancelled?: boolean;
};

export type ExamEdit = {
  id?: number;
  name?: string;
  is_active?: boolean;
  show_score?: boolean;
  show_percentile?: boolean;
  show_score_full?: boolean;
  validated_tribunal?: boolean;
  questions: QuestionEdit[];
};

export type AnswerReview = {
  question_id: number;
  question_label?: number | null;
  selected_option?: string | null;
  correct_option?: string | null;
  is_correct: boolean;
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
  result_type: string;
  created_at: string;
};

export type CreateOfficialResultPayload = {
  dni: string;
  apellido_1: string;
  apellido_2?: string | null;
  nombre: string;
  result_type: string;
};

export type EditOfficialResultPayload = {
  dni?: string;
  apellido_1?: string;
  apellido_2?: string | null;
  nombre?: string;
  result_type?: string;
};


export type AdminOfficialResultsResponse = {
  results: ExamOfficialResult[];
  total: number;
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
  answersReview: AnswerReview[] | null;
};

export type ExamResultPayload = {
  score: number | null;
  percentile: number | null;
  position: number | null;
  totalSubmissions: number | null;
  correctAnswers: number | null;
  totalQuestions: number | null;
  message: string;
  answersReview: AnswerReview[] | null;
};

export type ExamUiAction =
  | { type: 'RESET' }
  | { type: 'CHECK_START' }
  | { type: 'CHECK_SUCCESS'; payload: ExamResultPayload }
  | { type: 'CHECK_ERROR'; payload?: string | null }
  | { type: 'SUBMIT_SUCCESS'; payload: ExamResultPayload }
  | { type: 'SUBMIT_ERROR'; payload?: string | null };


