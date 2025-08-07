from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from db.database import get_db
from db.models import AdminUser, Exam, Question
from models.exam import ExamCreateWithQuestions, ExamEdit
from services.auth.auth_service import authenticate_admin, get_current_admin_user, get_password_hash, create_access_token
from models.admin import AdminCreate, AdminLogin, TokenResponse

router = APIRouter()

@router.post("/create-admin", status_code=201)
def create_admin(data: AdminCreate, db: Session = Depends(get_db)):
    # Verificar si ya existe un administrador
    if db.query(AdminUser).first():
        raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="Ya existe un administrador")

    # Crear el nuevo administrador
    hashed_password = get_password_hash(data.password)
    new_admin = AdminUser(username=data.username, password_hash=hashed_password)
    db.add(new_admin)
    db.commit()
    db.refresh(new_admin)
    return {"message": "Administrador creado con éxito"}


@router.post("/login", response_model=TokenResponse)
def login(data: AdminLogin, db: Session = Depends(get_db)):
    admin = authenticate_admin(db, data.username, data.password)
    if not admin:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Credenciales inválidas")

    token = create_access_token({"sub": admin.username})
    return {"access_token": token}

@router.post("/exams", status_code=201)
def create_exam_with_answers(
    exam_data: ExamCreateWithQuestions,
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user)
):
    existing = db.query(Exam).filter(Exam.name == exam_data.name).first()
    if existing:
        raise HTTPException(status_code=400, detail="Ya existe un examen con ese nombre")

    new_exam = Exam(
        name=exam_data.name,
        is_active=getattr(exam_data, 'is_active', False),
        show_response=getattr(exam_data, 'show_response', False),
        questions=[
            Question(correct_option=q.correct_option.upper())
            for q in exam_data.questions
        ]
    )

    db.add(new_exam)
    db.commit()
    db.refresh(new_exam)

    return {
        "id": new_exam.id,
        "name": new_exam.name,
        "questions_count": len(new_exam.questions)
    }

@router.put("/exams/{exam_id}/edit", status_code=200)
def edit_exam_with_answers(
    exam_id: int,
    exam_data: ExamEdit,
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user)
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
            id=q_data.id,  # incluye el id para que SQLAlchemy sepa si actualizar o crear
            correct_option=q_data.correct_option.upper(),
            exam=existing_exam  # ligamos al exam para la relación
        )
        updated_questions.append(q)

    existing_exam.questions = updated_questions

    db.commit()
    db.refresh(existing_exam)

    return {
        "id": existing_exam.id,
        "name": existing_exam.name,
        "questions_count": len(existing_exam.questions)
    }

@router.delete("/exams/{exam_id}/delete", status_code=200)
def delete_exam(
    exam_id: int,
    db: Session = Depends(get_db),
    admin: AdminUser = Depends(get_current_admin_user)
):
    exam = db.query(Exam).filter(Exam.id == exam_id).first()
    if not exam:
        raise HTTPException(status_code=404, detail="No existe un examen con ese ID")

    db.delete(exam)
    db.commit()

    return {"detail": f"Examen {exam_id} eliminado correctamente."}

@router.get("/check-token")
def check_token(user: AdminUser = Depends(get_current_admin_user)):
    return {"detail": "Token válido", "user": user.username}

@router.get("/exams/{exam_id}", response_model=ExamEdit)
def get_exam_by_id(
    exam_id: int,
    db: Session = Depends(get_db), 
    admin: AdminUser = Depends(get_current_admin_user)
):
    exam = db.query(Exam).filter(Exam.id == exam_id).first()
    return exam