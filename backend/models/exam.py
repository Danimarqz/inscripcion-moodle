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


class ExamUserBase(BaseModel):
    name: str
    surname: str
    email: EmailStr | None = None
    dni: str


class ExamUserOut(ExamUserBase):
    id: int

    class Config:
        from_attributes = True


class ExamOut(BaseModel):
    id: int
    name: str
    score: Optional[float] = None
    percentile: Optional[float] = None
    show_reponse: Optional[bool] = False

    class Config:
        from_attributes = True


class QuestionStubOut(BaseModel):
    id: int

    class Config:
        from_attributes = True


class QuestionCreate(BaseModel):
    correct_option: str  # 'A', 'B', 'C' o 'D'


class ExamCreateWithQuestions(BaseModel):
    name: str
    is_active: Optional[bool] = None
    show_response: Optional[bool] = None
    questions: List[QuestionCreate]


class QuestionEdit(BaseModel):
    id: Optional[int] = None
    correct_option: str


class ExamEdit(BaseModel):
    name: Optional[str] = None
    is_active: Optional[bool] = None
    show_response: Optional[bool] = None
    questions: List[QuestionEdit]


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
    score: float
    percentile: float


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
