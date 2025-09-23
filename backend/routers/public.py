from typing import List
from fastapi import APIRouter, Depends, HTTPException, Request
from sqlalchemy.orm import Session

from db.database import get_db
from db.models import Exam, UserExamSubmission
from models.exam import ExamOut, ExamSubmission, QuestionStubOut, SubmissionCheckRequest, SubmissionCheckResponse
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
    exams = db.query(Exam).filter(Exam.is_active).all()
    return exams

@router.get("/exams/{exam_id}/questions", response_model=List[QuestionStubOut])
def get_question_stubs(exam_id: int, db: Session = Depends(get_db)):
    exam = db.query(Exam).filter(Exam.id == exam_id).first()
    if not exam:
        raise HTTPException(status_code=404, detail="Examen no encontrado")

    return exam.questions

@router.post("/check_submission", response_model=SubmissionCheckResponse)
def check_submission(
    data: SubmissionCheckRequest,
    db: Session = Depends(get_db),
):
    submission = (
        db.query(UserExamSubmission)
        .filter_by(dni=data.dni, email=data.email, exam_id=data.exam_id)
        .first()
    )
    if not submission:
        raise HTTPException(status_code=404, detail="Submission not found")

    exam = db.query(Exam).filter(Exam.id == data.exam_id).first()
    if not exam or not exam.is_active or not exam.show_response:
        raise HTTPException(status_code=403, detail="Exam is not active or responses not visible")

    return SubmissionCheckResponse(
        score=submission.score,
        percentile=submission.percentile,
    )