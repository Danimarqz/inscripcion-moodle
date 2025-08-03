from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from db.database import SessionLocal, get_db
from db.models import AdminUser, Exam, Question
from models.exam import ExamCreateWithQuestions
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
    )
    db.add(new_exam)
    db.flush()  # para obtener new_exam.id

    for q in exam_data.questions:
        new_question = Question(
            exam_id=new_exam.id,
            text=q.text,           
            correct_option=q.correct_option.upper()
        )
        db.add(new_question)

    db.commit()
    db.refresh(new_exam)

    return {
        "id": new_exam.id,
        "name": new_exam.name,
        "questions_count": len(exam_data.questions)
    }
