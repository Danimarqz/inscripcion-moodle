from math import isclose
from typing import Dict, Iterable, List

from sqlalchemy.orm import Session, joinedload

from db.models import Exam, ExamUser, Question, UserAnswer, UserExamSubmission
from models.exam import ExamSubmission

DNI_LETTERS = "TRWAGMYFPDXBNJZSQVHLCKE"
VALID_OPTIONS = {"A", "B", "C", "D"}


def calculate_percentile(user_score: float, all_scores: Iterable[float | None]) -> float:
    scores = [score for score in all_scores if score is not None]
    if not scores:
        return 100.0

    epsilon = 1e-6
    count_less = sum(1 for score in scores if score < user_score - epsilon)
    count_equal = sum(
        1
        for score in scores
        if isclose(score, user_score, rel_tol=1e-9, abs_tol=epsilon)
    )
    percentile = (count_less + count_equal) / len(scores) * 100
    return round(percentile, 2)


def recalculate_percentiles(exam_id: int, db: Session, *, commit: bool = True) -> None:
    submissions = (
        db.query(UserExamSubmission)
        .filter(UserExamSubmission.exam_id == exam_id)
        .all()
    )
    if not submissions:
        return

    scores = [submission.score for submission in submissions if submission.score is not None]

    for submission in submissions:
        if submission.score is None:
            submission.percentile = 0.0
            continue

        if not scores:
            submission.percentile = 100.0
            continue

        submission.percentile = calculate_percentile(submission.score, scores)

    if commit:
        db.commit()
    else:
        db.flush()


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


def calculate_submission_score(
    active_questions: Iterable[Question],
    answers_by_question: Dict[int, str],
) -> float:
    active_list: List[Question] = [question for question in active_questions if question.is_active]
    total_active = len(active_list)
    if total_active == 0:
        raise ValueError("El examen no tiene preguntas activas configuradas")

    correct_count = sum(
        1
        for question in active_list
        if answers_by_question.get(question.id, "").upper()
        == question.correct_option.upper()
    )
    return correct_count / total_active * 100


def process_exam_submission(data: ExamSubmission, db: Session):
    normalized_email = data.email.lower()
    normalized_dni = normalize_dni(data.dni)

    if not validate_dni_nie(normalized_dni):
        raise Exception("DNI o NIE invalido")

    candidate = (
        db.query(ExamUser)
        .filter(ExamUser.dni == normalized_dni)
        .one_or_none()
    )

    if candidate and candidate.email:
        candidate.email = candidate.email.lower()

    if candidate is None:
        candidate = ExamUser(
            name=data.name.strip(),
            surname=data.surname.strip(),
            email=normalized_email,
            dni=normalized_dni,
        )
        db.add(candidate)
        db.flush()
    else:
        # Update main fields to keep consolidated data
        candidate.name = data.name.strip()
        candidate.surname = data.surname.strip()
        candidate.email = normalized_email
        db.flush()

    existing = (
        db.query(UserExamSubmission)
        .filter_by(user_id=candidate.id, exam_id=data.exam_id)
        .first()
    )
    if existing:
        raise Exception("Ya has enviado este examen")

    exam = (
        db.query(Exam)
        .options(joinedload(Exam.questions))
        .filter(Exam.id == data.exam_id)
        .first()
    )
    if not exam:
        raise Exception("Examen no encontrado")

    questions = exam.questions
    if not questions:
        raise Exception("El examen no tiene preguntas configuradas")

    questions_dict: Dict[int, Question] = {q.id: q for q in questions}
    answers_by_question: Dict[int, str] = {}

    for ans in data.answers:
        question = questions_dict.get(ans.question_id)
        if not question:
            raise Exception(f"Pregunta {ans.question_id} no encontrada")
        answers_by_question[ans.question_id] = validate_answer_option(ans.answer)

    try:
        score = calculate_submission_score(questions, answers_by_question)
    except ValueError as exc:
        raise Exception(str(exc))

    submission = UserExamSubmission(
        user_id=candidate.id,
        exam_id=data.exam_id,
        score=score,
        percentile=0,
    )
    db.add(submission)
    db.flush()

    for ans in data.answers:
        ua = UserAnswer(
            submission_id=submission.id,
            question_id=ans.question_id,
            answer=validate_answer_option(ans.answer),
        )
        db.add(ua)
    db.flush()

    recalculate_scores(data.exam_id, db, commit=False)
    db.commit()
    db.refresh(submission)

    if exam.show_response:
        return {
            "score": submission.score,
            "percentile": submission.percentile,
            "message": "Examen enviado correctamente",
        }
    return {"message": "Examen enviado correctamente"}


def recalculate_scores(exam_id: int, db: Session, *, commit: bool = True) -> None:
    exam = (
        db.query(Exam)
        .options(
            joinedload(Exam.questions),
            joinedload(Exam.submissions).joinedload(UserExamSubmission.answers),
        )
        .filter(Exam.id == exam_id)
        .first()
    )
    if not exam:
        return

    answers_by_submission: Dict[int, Dict[int, str]] = {}
    for submission in exam.submissions:
        answers_by_submission[submission.id] = {
            answer.question_id: answer.answer.upper()
            for answer in submission.answers
        }

    for submission in exam.submissions:
        answer_map = answers_by_submission.get(submission.id, {})
        try:
            submission.score = calculate_submission_score(exam.questions, answer_map)
        except ValueError:
            submission.score = 0.0

    db.flush()
    recalculate_percentiles(exam_id, db, commit=commit)
