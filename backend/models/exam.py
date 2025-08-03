from pydantic import BaseModel, EmailStr
from typing import List, Literal

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
    questions: List[QuestionCreate]