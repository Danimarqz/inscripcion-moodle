from typing import List
from fastapi import APIRouter, Depends, HTTPException, Request
from sqlalchemy.orm import Session

from db.database import SessionLocal, get_db
from db.models import Exam
from models.exam import AnswerSubmission, ExamOut, ExamSubmission, QuestionStubOut
from rate_limiter import check_rate_limit
from services.exam.submit_exam import process_exam_submission

router = APIRouter()

@router.post("/submit-exam")
def submit_exam(request: Request, data: ExamSubmission, db: Session = Depends(get_db)):
    check_rate_limit(request)
    try:
        result = process_exam_submission(data, db)
        return result
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@router.get("/exams", response_model=List[ExamOut])
def get_exams(db: Session = Depends(get_db)):
    exams = db.query(Exam).all()
    return exams

@router.get("/exams/{exam_id}/questions", response_model=List[QuestionStubOut])
def get_question_stubs(exam_id: int, db: Session = Depends(get_db)):
    exam = db.query(Exam).filter(Exam.id == exam_id).first()
    if not exam:
        raise HTTPException(status_code=404, detail="Examen no encontrado")

    return exam.questions
