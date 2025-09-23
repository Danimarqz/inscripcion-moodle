from typing import Dict, List

from sqlalchemy.orm import Session

from db.models import Exam, Question, UserAnswer, UserExamSubmission
from models.exam import ExamSubmission

DNI_LETTERS = "TRWAGMYFPDXBNJZSQVHLCKE"
VALID_OPTIONS = {"A", "B", "C", "D"}


def calculate_percentile(user_score: float, all_scores: List[float]) -> float:
    if not all_scores:
        return 100.0
    count_less = sum(1 for score in all_scores if score < user_score)
    return round(100 * count_less / len(all_scores), 2)


def normalize_dni(value: str) -> str:
    return value.strip().upper()


def validate_dni_nie(value: str) -> bool:
    if not value:
        return False
    upper = value.strip().upper()
    if len(upper) != 9:
        return False

    letter = upper[-1]
    if not letter.isalpha():
        return False

    if upper[0] in {"X", "Y", "Z"}:
        prefix_map: Dict[str, str] = {"X": "0", "Y": "1", "Z": "2"}
        numeric_part = prefix_map[upper[0]] + upper[1:-1]
    else:
        numeric_part = upper[:-1]

    if not numeric_part.isdigit():
        return False

    expected_letter = DNI_LETTERS[int(numeric_part) % 23]
    return expected_letter == letter


def validate_answer_option(option: str) -> str:
    upper = option.upper()
    if upper not in VALID_OPTIONS:
        raise Exception("Opcion de respuesta no valida")
    return upper


def process_exam_submission(data: ExamSubmission, db: Session):
    normalized_email = data.email.lower()
    normalized_dni = normalize_dni(data.dni)

    if not validate_dni_nie(normalized_dni):
        raise Exception("DNI o NIE invalido")

    existing = (
        db.query(UserExamSubmission)
        .filter_by(email=normalized_email, exam_id=data.exam_id)
        .first()
    )
    if existing:
        raise Exception("Ya has enviado este examen")

    questions = db.query(Question).filter(Question.exam_id == data.exam_id).all()
    if not questions:
        raise Exception("El examen no tiene preguntas configuradas")

    questions_dict: Dict[int, Question] = {q.id: q for q in questions}
    exam = db.query(Exam).filter(Exam.id == data.exam_id).first()
    if not exam:
        raise Exception("Examen no encontrado")

    correct_count = 0
    for ans in data.answers:
        question = questions_dict.get(ans.question_id)
        if not question:
            raise Exception(f"Pregunta {ans.question_id} no encontrada")
        answer_value = validate_answer_option(ans.answer)
        if answer_value == question.correct_option.upper():
            correct_count += 1

    score = correct_count / len(questions) * 100

    submission = UserExamSubmission(
        email=normalized_email,
        dni=normalized_dni,
        name=data.name.strip(),
        surname=data.surname.strip(),
        exam_id=data.exam_id,
        score=score,
        percentile=0,
    )
    db.add(submission)
    db.commit()
    db.refresh(submission)

    for ans in data.answers:
        ua = UserAnswer(
            submission_id=submission.id,
            question_id=ans.question_id,
            answer=validate_answer_option(ans.answer),
        )
        db.add(ua)
    db.commit()

    all_scores = [s.score for s in db.query(UserExamSubmission).filter_by(exam_id=data.exam_id).all()]
    percentile = calculate_percentile(score, all_scores)
    submission.percentile = percentile
    db.commit()

    if exam.show_response:
        return {"score": score, "percentile": percentile, "message": "Examen enviado correctamente"}
    return {"message": "Examen enviado correctamente"}
