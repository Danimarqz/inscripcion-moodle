from typing import List, Optional

from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy.orm import Session

from db.database import get_db
from db.models import AdminUser, Exam, Question, UserAnswer, UserExamSubmission
from models.admin import AdminCreate, AdminLogin, TokenResponse
from models.exam import (
    AdminSubmissionOut,
    AdminSubmissionUpdate,
    ExamCreateWithQuestions,
    ExamEdit,
    ExamOut,
)
from services.auth.auth_service import (
    authenticate_admin,
    create_access_token,
    get_current_admin_user,
    get_password_hash,
)
from services.exam.submit_exam import (
    normalize_dni,
    recalculate_percentiles,
    validate_answer_option,
    validate_dni_nie,
)

router = APIRouter()


@router.get("/exams", response_model=List[ExamOut])
def get_exams(
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user),
):
    exams = db.query(Exam).all()
    return exams


@router.post("/create-admin", status_code=201)
def create_admin(data: AdminCreate, db: Session = Depends(get_db)):
    if db.query(AdminUser).first():
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Ya existe un administrador",
        )

    hashed_password = get_password_hash(data.password)
    new_admin = AdminUser(username=data.username, password_hash=hashed_password)
    db.add(new_admin)
    db.commit()
    db.refresh(new_admin)
    return {"message": "Administrador creado con exito"}


@router.post("/login", response_model=TokenResponse)
def login(data: AdminLogin, db: Session = Depends(get_db)):
    admin = authenticate_admin(db, data.username, data.password)
    if not admin:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Credenciales invalidas",
        )

    token = create_access_token({"sub": admin.username})
    return {"access_token": token}


@router.post("/exams", status_code=201)
def create_exam_with_answers(
    exam_data: ExamCreateWithQuestions,
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user),
):
    existing = db.query(Exam).filter(Exam.name == exam_data.name).first()
    if existing:
        raise HTTPException(status_code=400, detail="Ya existe un examen con ese nombre")

    new_exam = Exam(
        name=exam_data.name,
        is_active=getattr(exam_data, "is_active", False),
        show_response=getattr(exam_data, "show_response", False),
        questions=[
            Question(correct_option=q.correct_option.upper())
            for q in exam_data.questions
        ],
    )

    db.add(new_exam)
    db.commit()
    db.refresh(new_exam)

    return {
        "id": new_exam.id,
        "name": new_exam.name,
        "questions_count": len(new_exam.questions),
    }


@router.put("/exams/{exam_id}/edit", status_code=200)
def edit_exam_with_answers(
    exam_id: int,
    exam_data: ExamEdit,
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user),
):
    existing_exam = db.query(Exam).filter(Exam.id == exam_id).first()
    if not existing_exam:
        raise HTTPException(status_code=404, detail="No existe un examen con ese ID")

    if exam_data.name is not None:
        existing_exam.name = exam_data.name
    if exam_data.is_active is not None:
        existing_exam.is_active = exam_data.is_active
    if exam_data.show_response is not None:
        existing_exam.show_response = exam_data.show_response

    updated_questions = []
    for q_data in exam_data.questions:
        q = Question(
            id=q_data.id,
            correct_option=q_data.correct_option.upper(),
            exam=existing_exam,
        )
        updated_questions.append(q)

    existing_exam.questions = updated_questions

    db.commit()
    db.refresh(existing_exam)

    return {
        "id": existing_exam.id,
        "name": existing_exam.name,
        "questions_count": len(existing_exam.questions),
    }


@router.delete("/exams/{exam_id}/delete", status_code=200)
def delete_exam(
    exam_id: int,
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user),
):
    exam = db.query(Exam).filter(Exam.id == exam_id).first()
    if not exam:
        raise HTTPException(status_code=404, detail="No existe un examen con ese ID")

    db.delete(exam)
    db.commit()

    return {"detail": f"Examen {exam_id} eliminado correctamente."}


@router.get("/check-token")
def check_token(user: AdminUser = Depends(get_current_admin_user)):
    return {"detail": "Token valido", "user": user.username}


@router.get("/exams/{exam_id}", response_model=ExamEdit)
def get_exam_by_id(
    exam_id: int,
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user),
):
    exam = db.query(Exam).filter(Exam.id == exam_id).first()
    return exam


@router.get("/results", response_model=List[AdminSubmissionOut])
def get_user_results(
    exam_id: Optional[int] = Query(None, description="Identificador del examen"),
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user),
):
    if exam_id is None:
        return []

    results = (
        db.query(UserExamSubmission)
        .filter(UserExamSubmission.exam_id == exam_id)
        .order_by(UserExamSubmission.submitted_at.desc())
        .all()
    )
    return results


@router.put("/results/{submission_id}", response_model=AdminSubmissionOut)
def update_submission(
    submission_id: int,
    data: AdminSubmissionUpdate,
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user),
):
    submission = db.query(UserExamSubmission).filter(UserExamSubmission.id == submission_id).first()
    if not submission:
        raise HTTPException(status_code=404, detail="Intento no encontrado")

    normalized_dni = normalize_dni(data.dni)
    if not validate_dni_nie(normalized_dni):
        raise HTTPException(status_code=400, detail="DNI o NIE invalido")

    submission.dni = normalized_dni
    submission.email = data.email.lower()
    submission.name = data.name.strip()
    submission.surname = data.surname.strip()

    questions = db.query(Question).filter(Question.exam_id == submission.exam_id).all()
    if not questions:
        raise HTTPException(status_code=400, detail="El examen asociado no tiene preguntas")

    question_map = {q.id: q for q in questions}
    provided_ids = set()
    existing_answers = {answer.question_id: answer for answer in submission.answers}

    for ans in data.answers:
        question = question_map.get(ans.question_id)
        if not question:
            raise HTTPException(
                status_code=400,
                detail=f"La pregunta {ans.question_id} no pertenece al examen",
            )
        try:
            normalized_answer = validate_answer_option(ans.answer)
        except Exception as exc:
            raise HTTPException(status_code=400, detail=str(exc))

        provided_ids.add(ans.question_id)
        if ans.question_id in existing_answers:
            existing_answers[ans.question_id].answer = normalized_answer
        else:
            db.add(
                UserAnswer(
                    submission=submission,
                    question_id=ans.question_id,
                    answer=normalized_answer,
                )
            )

    for question_id, answer_obj in list(existing_answers.items()):
        if question_id not in provided_ids:
            db.delete(answer_obj)

    db.flush()
    answer_map = {answer.question_id: answer.answer.upper() for answer in submission.answers}
    total_questions = len(question_map)
    if total_questions:
        correct_count = sum(
            1
            for question in question_map.values()
            if answer_map.get(question.id) == question.correct_option.upper()
        )
        submission.score = correct_count / total_questions * 100
    else:
        submission.score = 0.0

    db.flush()
    recalculate_percentiles(submission.exam_id, db, commit=False)

    db.commit()
    db.refresh(submission)
    return submission


@router.delete("/results/{submission_id}", status_code=200)
def delete_submission(
    submission_id: int,
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user),
):
    submission = db.query(UserExamSubmission).filter(UserExamSubmission.id == submission_id).first()
    if not submission:
        raise HTTPException(status_code=404, detail="Intento no encontrado")

    db.delete(submission)
    db.commit()
    return {"detail": "Intento eliminado correctamente."}






