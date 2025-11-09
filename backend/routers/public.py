from typing import List
from fastapi import APIRouter, Depends, HTTPException, Request
from sqlalchemy.orm import Session

from db.database import get_db
from db.models import Exam, ExamUser, UserExamSubmission
from models.exam import ExamOut, ExamSubmission, QuestionStubOut, SubmissionCheckRequest, SubmissionCheckResponse
from rate_limiter import check_rate_limit
from services.exam.submit_exam import (
    build_submission_payload,
    normalize_dni,
    process_exam_submission,
)

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

    return sorted(
        exam.questions,
        key=lambda question: (question.name if isinstance(question.name, int) else float("inf"), question.id),
    )

@router.post("/check_submission", response_model=SubmissionCheckResponse)
def check_submission(
    data: SubmissionCheckRequest,
    db: Session = Depends(get_db),
):
    submission = (
        db.query(UserExamSubmission)
        .join(ExamUser)
        .filter(
            ExamUser.dni == normalize_dni(data.dni),
            ExamUser.email == data.email.lower(),
            UserExamSubmission.exam_id == data.exam_id,
        )
        .first()
    )
    if not submission:
        raise HTTPException(status_code=404, detail="Submission not found")

    exam = db.query(Exam).filter(Exam.id == data.exam_id).first()
    if not exam or not exam.is_active:
        raise HTTPException(status_code=403, detail="Exam is not active or responses not visible")

    if not any([exam.show_score, exam.show_percentile, exam.show_score_full]):
        raise HTTPException(
            status_code=403,
            detail="Los resultados no están disponibles para este examen",
        )

    payload = build_submission_payload(
        exam=exam,
        submission=submission,
        db=db,
        message="Ya has enviado este examen anteriormente",
    )
    return SubmissionCheckResponse(**payload)
