from datetime import datetime
from typing import List, Literal, Optional

from pydantic import BaseModel, EmailStr, computed_field


class AnswerSubmission(BaseModel):
    question_id: int
    answer: Literal["A", "B", "C", "D"]


class ExamSubmission(BaseModel):
    email: EmailStr
    dni: str
    name: str
    surname: str
    exam_id: int
    answers: List[AnswerSubmission]
    accepts_marketing: bool = False


class ExamUserBase(BaseModel):
    name: str
    surname: str
    email: EmailStr | None = None
    dni: str
    accepts_marketing: bool = False


class ExamUserOut(ExamUserBase):
    id: int

    class Config:
        from_attributes = True


class ExamOut(BaseModel):
    id: int
    name: str
    score: Optional[float] = None
    percentile: Optional[float] = None
    show_score: bool = False
    show_percentile: bool = False
    show_score_full: bool = False
    validated_tribunal: bool = False

    class Config:
        from_attributes = True


class QuestionStubOut(BaseModel):
    id: int
    name: int
    is_active: bool
    is_cancelled: bool = False

    class Config:
        from_attributes = True


class QuestionCreate(BaseModel):
    correct_option: str  # 'A', 'B', 'C' o 'D'
    is_active: bool = True
    is_cancelled: bool = False
    name: Optional[int] = None


class ExamCreateWithQuestions(BaseModel):
    name: str
    is_active: Optional[bool] = None
    show_score: Optional[bool] = None
    show_percentile: Optional[bool] = None
    show_score_full: Optional[bool] = None
    validated_tribunal: Optional[bool] = None
    questions: List[QuestionCreate]


class QuestionEdit(BaseModel):
    id: Optional[int] = None
    correct_option: str
    is_active: bool = True
    is_cancelled: bool = False
    name: Optional[int] = None


class ExamEdit(BaseModel):
    name: Optional[str] = None
    is_active: Optional[bool] = None
    show_score: Optional[bool] = None
    show_percentile: Optional[bool] = None
    show_score_full: Optional[bool] = None
    validated_tribunal: Optional[bool] = None
    questions: List[QuestionEdit]


class AnswerReview(BaseModel):
    question_id: int
    question_label: Optional[int] = None
    selected_option: Optional[str] = None
    correct_option: Optional[str] = None
    is_correct: bool = False

    class Config:
        from_attributes = True


class AnswerDetail(BaseModel):
    id: int
    question_id: int
    answer: Literal["A", "B", "C", "D"]

    class Config:
        from_attributes = True


class UserExamSubmission(BaseModel):
    id: int
    exam_id: int
    user: ExamUserOut
    score: Optional[float] = None
    percentile: Optional[float] = None
    submitted_at: datetime

    class Config:
        from_attributes = True

    @computed_field
    def email(self) -> EmailStr | None:
        return self.user.email

    @computed_field
    def dni(self) -> str:
        return self.user.dni

    @computed_field
    def name(self) -> str:
        return self.user.name

    @computed_field
    def surname(self) -> str:
        return self.user.surname

    @computed_field
    def accepts_marketing(self) -> bool:
        return bool(getattr(self.user, "accepts_marketing", False))


class AdminSubmissionOut(UserExamSubmission):
    answers: List[AnswerDetail]


class AdminSubmissionUpdate(BaseModel):
    name: str
    surname: str
    email: EmailStr
    dni: str
    answers: List[AnswerSubmission]


class SubmissionCheckRequest(BaseModel):
    dni: str
    email: EmailStr
    exam_id: int


class SubmissionCheckResponse(BaseModel):
    score: Optional[float] = None
    percentile: float | None = None
    position: Optional[int] = None
    total_submissions: Optional[int] = None
    correct_answers: Optional[int] = None
    total_questions: Optional[int] = None
    message: Optional[str] = None
    answers_review: Optional[List[AnswerReview]] = None


class ExamOfficialResultOut(BaseModel):
    id: int
    exam_id: int
    user: ExamUserOut | None = None
    dni_masked: str
    apellido_1: str
    apellido_2: Optional[str]
    nombre: str
    created_at: datetime

    class Config:
        from_attributes = True
