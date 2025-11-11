import os
from pathlib import Path
from tempfile import NamedTemporaryFile
from typing import Callable, Iterable, List, Optional, Union

from fastapi import APIRouter, Depends, File, HTTPException, Query, UploadFile, status
from sqlalchemy.orm import Session, joinedload

from db.database import get_db
from db.models import (
    AdminUser,
    Exam,
    ExamOfficialResult,
    ExamUser,
    Question,
    UserAnswer,
    UserExamSubmission,
)
from models.admin import AdminCreate, AdminLogin, TokenResponse
from models.exam import (
    AdminSubmissionOut,
    AdminSubmissionUpdate,
    ExamCreateWithQuestions,
    ExamEdit,
    ExamOfficialResultOut,
    ExamOut,
)
from services.auth.auth_service import (
    authenticate_admin,
    create_access_token,
    get_current_admin_user,
    get_password_hash,
)
from services.cache import invalidate_exam_cache, invalidate_check_cache_for_exam
from services.exam.submit_exam import (
    normalize_dni,
    recalculate_scores_bulk,
    validate_answer_option,
    validate_dni_nie,
)
from services.exam.exam_results_importer import (
    ExamResultImportError,
    import_official_results_from_pdf,
)

router = APIRouter()


def _normalize_question_name(value: Optional[Union[int, str]]) -> Optional[int]:
    if value is None:
        return None
    if isinstance(value, int):
        return value if value > 0 else None
    text_value = str(value).strip()
    if not text_value:
        return None
    try:
        parsed = int(text_value)
    except ValueError:
        return None
    return parsed if parsed > 0 else None


def _build_question_name_generator(existing_names: Iterable[int]) -> Callable[[Optional[Union[int, str]]], int]:
    used_names = {name for name in existing_names if isinstance(name, int) and name > 0}
    next_counter = max(used_names) + 1 if used_names else 1

    def _reserve(preferred: Optional[Union[int, str]]) -> int:
        nonlocal next_counter
        candidate = _normalize_question_name(preferred)
        if candidate and candidate not in used_names:
            used_names.add(candidate)
            return candidate

        while True:
            candidate = next_counter
            next_counter += 1
            if candidate not in used_names:
                used_names.add(candidate)
                return candidate

    return _reserve


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

    if not exam_data.questions:
        raise HTTPException(status_code=400, detail="El examen debe tener al menos una pregunta")

    question_models = []
    name_reserver = _build_question_name_generator([])
    active_questions = 0
    for q in exam_data.questions:
        try:
            normalized_option = validate_answer_option(q.correct_option)
        except Exception as exc:
            raise HTTPException(status_code=400, detail=str(exc))
        is_active = bool(getattr(q, "is_active", True))
        is_cancelled = bool(getattr(q, "is_cancelled", False))
        if is_active and not is_cancelled:
            active_questions += 1
        question_models.append(
            Question(
                name=name_reserver(getattr(q, "name", None)),
                correct_option=normalized_option,
                is_active=is_active,
                is_cancelled=is_cancelled,
            )
        )

    if active_questions == 0:
        raise HTTPException(
            status_code=400,
            detail="El examen debe tener al menos una pregunta activa no anulada",
        )

    new_exam = Exam(
        name=exam_data.name,
        is_active=getattr(exam_data, "is_active", False),
        show_score=bool(getattr(exam_data, "show_score", False)),
        show_percentile=bool(getattr(exam_data, "show_percentile", False)),
        show_score_full=bool(getattr(exam_data, "show_score_full", False)),
        validated_tribunal=bool(getattr(exam_data, "validated_tribunal", False)),
        questions=question_models,
    )

    db.add(new_exam)
    db.commit()
    db.refresh(new_exam)
    invalidate_exam_cache(new_exam.id)

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
    if exam_data.show_score is not None:
        existing_exam.show_score = exam_data.show_score
    if exam_data.show_percentile is not None:
        existing_exam.show_percentile = exam_data.show_percentile
    if exam_data.show_score_full is not None:
        existing_exam.show_score_full = exam_data.show_score_full
    if exam_data.validated_tribunal is not None:
        existing_exam.validated_tribunal = exam_data.validated_tribunal

    if not exam_data.questions:
        raise HTTPException(status_code=400, detail="El examen debe tener al menos una pregunta")

    payload_ids = {q.id for q in exam_data.questions if q.id is not None}
    existing_map = {question.id: question for question in existing_exam.questions}

    for question in list(existing_exam.questions):
        if question.id not in payload_ids:
            db.delete(question)

    name_reserver = _build_question_name_generator([question.name for question in existing_exam.questions])

    for q_data in exam_data.questions:
        try:
            normalized_option = validate_answer_option(q_data.correct_option)
        except Exception as exc:
            raise HTTPException(status_code=400, detail=str(exc))
        is_active = bool(getattr(q_data, "is_active", True))
        is_cancelled = bool(getattr(q_data, "is_cancelled", False))

        if q_data.id is not None:
            question = existing_map.get(q_data.id)
            if not question:
                raise HTTPException(
                    status_code=400,
                    detail=f"La pregunta {q_data.id} no pertenece al examen",
                )
            question.correct_option = normalized_option
            question.is_active = is_active
            question.is_cancelled = is_cancelled
        else:
            db.add(
                Question(
                    exam=existing_exam,
                    name=name_reserver(getattr(q_data, "name", None)),
                    correct_option=normalized_option,
                    is_active=is_active,
                    is_cancelled=is_cancelled,
                )
            )

    db.flush()
    active_total = sum(
        1 for question in existing_exam.questions if question.is_active and not question.is_cancelled
    )
    if active_total == 0:
        raise HTTPException(
            status_code=400,
            detail="El examen debe tener al menos una pregunta activa no anulada",
        )

    recalculate_scores_bulk(existing_exam.id, db)
    db.commit()
    db.refresh(existing_exam)
    invalidate_exam_cache(existing_exam.id)

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
    invalidate_exam_cache(exam_id)

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
        .options(
            joinedload(UserExamSubmission.user),
            joinedload(UserExamSubmission.answers),
        )
        .filter(UserExamSubmission.exam_id == exam_id)
        .order_by(UserExamSubmission.submitted_at.desc())
        .all()
    )
    return results


@router.get(
    "/exams/{exam_id}/results/official",
    response_model=List[ExamOfficialResultOut],
)
def list_official_results(
    exam_id: int,
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user),
):
    results = (
        db.query(ExamOfficialResult)
        .options(joinedload(ExamOfficialResult.user))
        .filter(ExamOfficialResult.exam_id == exam_id)
        .order_by(
            ExamOfficialResult.apellido_1.asc(),
            ExamOfficialResult.apellido_2.asc(),
            ExamOfficialResult.nombre.asc(),
        )
        .all()
    )
    return results


@router.post("/exams/{exam_id}/results/import")
async def import_official_results(
    exam_id: int,
    file: UploadFile = File(...),
    replace_existing: bool = Query(
        True,
        description="Eliminar resultados oficiales previos antes de importar",
    ),
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user),
):
    if file.content_type not in {"application/pdf", "application/octet-stream"}:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="El archivo debe ser un PDF",
        )

    tmp_path = None
    try:
        with NamedTemporaryFile(delete=False, suffix=".pdf") as tmp:
            content = await file.read()
            tmp.write(content)
            tmp_path = Path(tmp.name)

        stats = import_official_results_from_pdf(
            db=db,
            exam_id=exam_id,
            pdf_path=tmp_path,
            replace_existing=replace_existing,
        )
    except ExamResultImportError as exc:
        raise HTTPException(status_code=400, detail=str(exc))
    finally:
        if tmp_path and tmp_path.exists():
            try:
                os.unlink(tmp_path)
            except OSError:
                pass

    return {
        "exam_id": stats.exam_id,
        "total_rows": stats.total_rows,
        "imported_results": stats.imported_results,
        "created_users": stats.created_users,
        "updated_users": stats.updated_users,
    }


@router.put("/results/{submission_id}", response_model=AdminSubmissionOut)
def update_submission(
    submission_id: int,
    data: AdminSubmissionUpdate,
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user),
):
    submission = (
        db.query(UserExamSubmission)
        .options(joinedload(UserExamSubmission.user))
        .filter(UserExamSubmission.id == submission_id)
        .first()
    )
    if not submission:
        raise HTTPException(status_code=404, detail="Intento no encontrado")

    normalized_dni = normalize_dni(data.dni)
    if not validate_dni_nie(normalized_dni):
        raise HTTPException(status_code=400, detail="DNI o NIE invalido")

    candidate = submission.user

    duplicate_user = (
        db.query(ExamUser)
        .filter(ExamUser.dni == normalized_dni, ExamUser.id != candidate.id)
        .first()
    )
    if duplicate_user:
        raise HTTPException(status_code=400, detail="DNI duplicado para otro usuario")

    candidate.dni = normalized_dni
    candidate.email = data.email.lower()
    candidate.name = data.name.strip()
    candidate.surname = data.surname.strip()

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
    recalculate_scores_bulk(submission.exam_id, db, commit=False)

    db.commit()
    db.refresh(submission)
    invalidate_check_cache_for_exam(submission.exam_id)
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
    invalidate_check_cache_for_exam(submission.exam_id)
    return {"detail": "Intento eliminado correctamente."}






