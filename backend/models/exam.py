from pydantic import BaseModel, EmailStr
from typing import List, Literal, Optional

class AnswerSubmission(BaseModel):
    question_id: int
    answer: Literal["A", "B", "C", "D"]

class ExamSubmission(BaseModel):
    email: EmailStr
    dni: str
    exam_id: int
    answers: List[AnswerSubmission]

class ExamOut(BaseModel):
    id: int
    name: str
    score: Optional[float] = None
    percentile: Optional[float] = None

    class Config:
        from_attributes = True

class QuestionStubOut(BaseModel):
    id: int

    class Config:
        from_attributes = True

class QuestionCreate(BaseModel):
    text: str
    correct_option: str  # 'A', 'B', 'C' o 'D'

class ExamCreateWithQuestions(BaseModel):
    name: str
    is_active: Optional[bool] = None
    show_response: Optional[bool] = None
    questions: List[QuestionCreate]

class QuestionEdit(BaseModel):
    id: Optional[int] = None
    text: Optional[str] = None
    correct_option: str

class ExamEdit(BaseModel):
    name: Optional[str] = None
    is_active: Optional[bool] = None
    show_response: Optional[bool] = None
    questions: List[QuestionEdit]
