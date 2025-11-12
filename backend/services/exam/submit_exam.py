import logging
from dataclasses import dataclass
from math import isclose
from typing import Any, Dict, Iterable, List, Optional, Tuple

from sqlalchemy import func, text
from sqlalchemy.orm import Session, joinedload

from db.models import Exam, ExamUser, Question, UserAnswer, UserExamSubmission
from models.exam import ExamSubmission
from services.cache import invalidate_check_cache_for_exam

DNI_LETTERS = "TRWAGMYFPDXBNJZSQVHLCKE"
VALID_OPTIONS = {"A", "B", "C", "D"}


logger = logging.getLogger(__name__)


@dataclass
class ScoreBreakdown:
    score: float
    correct_answers: int
    total_questions: int


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
    percentile_sql = text(
        """
        UPDATE user_exam_submission AS u
        JOIN (
            SELECT
                id,
                exam_id,
                ROUND(
                    CUME_DIST() OVER (PARTITION BY exam_id ORDER BY score) * 100,
                    2
                ) AS pct
            FROM user_exam_submission
            WHERE exam_id = :exam_id AND score IS NOT NULL
        ) ranked ON ranked.id = u.id
        SET u.percentile = ranked.pct
        WHERE u.exam_id = :exam_id
        """
    )

    db.execute(percentile_sql, {"exam_id": exam_id})

    if commit:
        db.commit()


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


def calculate_score_breakdown(
    active_questions: Iterable[Question],
    answers_by_question: Dict[int, str],
) -> ScoreBreakdown:
    active_list: List[Question] = [
        question
        for question in active_questions
        if question.is_active and not question.is_cancelled
    ]
    total_active = len(active_list)
    if total_active == 0:
        raise ValueError("El examen no tiene preguntas activas no anuladas configuradas")

    correct_count = sum(
        1
        for question in active_list
        if answers_by_question.get(question.id, "").upper()
        == question.correct_option.upper()
    )
    score = correct_count / total_active * 100
    return ScoreBreakdown(score=score, correct_answers=correct_count, total_questions=total_active)


def calculate_submission_score(
    active_questions: Iterable[Question],
    answers_by_question: Dict[int, str],
) -> float:
    return calculate_score_breakdown(active_questions, answers_by_question).score


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
            accepts_marketing=data.accepts_marketing,
        )
        db.add(candidate)
        db.flush()
    else:
        # Update main fields to keep consolidated data
        candidate.name = data.name.strip()
        candidate.surname = data.surname.strip()
        candidate.email = normalized_email
        candidate.accepts_marketing = data.accepts_marketing
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
        breakdown = calculate_score_breakdown(questions, answers_by_question)
    except ValueError as exc:
        raise Exception(str(exc))

    submission = UserExamSubmission(
        user_id=candidate.id,
        exam_id=data.exam_id,
        score=breakdown.score,
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
    invalidate_check_cache_for_exam(exam.id)

    return build_submission_payload(
        exam=exam,
        submission=submission,
        db=db,
        message="Examen enviado correctamente",
        score_breakdown=breakdown,
    )


def recalculate_scores(exam_id: int, db: Session, *, commit: bool = True) -> None:
    recalculate_percentiles(exam_id, db, commit=commit)

def recalculate_scores_bulk(exam_id: int, db: Session) -> None:
    """Recalcula todas las puntuaciones de un examen directamente en SQL."""
    db.execute(text("""
        UPDATE user_exam_submission AS u
        JOIN (
            SELECT
                ua.submission_id,
                ROUND(SUM(CASE
                    WHEN UPPER(ua.answer) = q.correct_option AND q.is_active = 1 AND q.is_cancelled = 0
                    THEN 1 ELSE 0 END) / COUNT(q.id) * 100, 2) AS score
            FROM user_answer ua
            JOIN question q ON q.id = ua.question_id
            WHERE q.exam_id = :exam_id
            GROUP BY ua.submission_id
        ) AS t ON u.id = t.submission_id
        SET u.score = t.score
        WHERE u.exam_id = :exam_id
    """), {"exam_id": exam_id})

    # Actualiza percentiles tras recalcular
    recalculate_percentiles(exam_id, db, commit=False)

def fetch_score_breakdown_from_db(
    *, exam_id: int, submission_id: int, db: Session
) -> Optional[ScoreBreakdown]:
    logger.debug(
        "Fetching score breakdown from DB for exam_id=%s submission_id=%s",
        exam_id,
        submission_id,
    )
    questions = db.query(Question).filter(Question.exam_id == exam_id).all()
    if not questions:
        logger.warning(
            "Cannot compute score breakdown: exam_id=%s has no questions", exam_id
        )
        return None

    answers = db.query(UserAnswer).filter(UserAnswer.submission_id == submission_id).all()
    answers_map = {answer.question_id: answer.answer.upper() for answer in answers}

    try:
        breakdown = calculate_score_breakdown(questions, answers_map)
        logger.debug(
            "Recovered breakdown for submission_id=%s -> correct=%s total=%s",
            submission_id,
            breakdown.correct_answers,
            breakdown.total_questions,
        )
        return breakdown
    except ValueError:
        logger.exception(
            "Failed to recalculate score breakdown for submission_id=%s", submission_id
        )
        return None


def get_submission_position_data(
    submission: UserExamSubmission,
    db: Session,
) -> Tuple[Optional[int], int]:
    total = (
        db.query(func.count(UserExamSubmission.id))
        .filter(UserExamSubmission.exam_id == submission.exam_id)
        .scalar()
    )
    total_submissions = int(total or 0)
    if total_submissions == 0 or submission.score is None:
        return None, total_submissions

    better = (
        db.query(func.count(UserExamSubmission.id))
        .filter(
            UserExamSubmission.exam_id == submission.exam_id,
            UserExamSubmission.score.isnot(None),
            UserExamSubmission.score > submission.score,
        )
        .scalar()
    )
    better_count = int(better or 0)
    return better_count + 1, total_submissions


def build_answers_review(
    *,
    exam: Exam,
    submission: UserExamSubmission,
    db: Session,
) -> List[Dict[str, Any]]:
    questions: List[Question] = list(getattr(exam, "questions", []))
    if not questions:
        questions = db.query(Question).filter(Question.exam_id == exam.id).all()

    answers: List[UserAnswer] = list(getattr(submission, "answers", []))
    if not answers:
        answers = db.query(UserAnswer).filter(UserAnswer.submission_id == submission.id).all()

    answers_map = {answer.question_id: answer.answer.upper() for answer in answers}

    def sort_key(question: Question) -> Tuple[float, int]:
        name_value = question.name if isinstance(question.name, int) else float("inf")
        return (float(name_value), question.id)

    review: List[Dict[str, Any]] = []
    for question in sorted(questions, key=sort_key):
        if not question.is_active or getattr(question, "is_cancelled", False):
            continue
        selected = answers_map.get(question.id)
        correct = question.correct_option.upper() if question.correct_option else None
        review.append(
            {
                "question_id": question.id,
                "question_label": question.name if isinstance(question.name, int) else None,
                "selected_option": selected,
                "correct_option": correct,
                "is_correct": bool(selected and correct and selected == correct),
            }
        )
    return review


def build_submission_payload(
    *,
    exam: Exam,
    submission: UserExamSubmission,
    db: Session,
    message: str,
    score_breakdown: Optional[ScoreBreakdown] = None,
) -> Dict[str, Any]:
    logger.debug(
        "Building submission payload exam_id=%s submission_id=%s flags(score=%s, percentile=%s, score_full=%s)",
        exam.id,
        submission.id,
        exam.show_score,
        exam.show_percentile,
        exam.show_score_full,
    )
    payload: Dict[str, Any] = {"message": message}
    payload["score"] = submission.score if exam.show_score else None

    if exam.show_percentile:
        payload["percentile"] = submission.percentile
        position, total = get_submission_position_data(submission, db)
        payload["position"] = position
        payload["total_submissions"] = total
    else:
        payload["percentile"] = None
        payload["position"] = None
        payload["total_submissions"] = None

    if exam.show_score_full:
        breakdown = score_breakdown or fetch_score_breakdown_from_db(
            exam_id=exam.id,
            submission_id=submission.id,
            db=db,
        )
        if breakdown:
            payload["correct_answers"] = breakdown.correct_answers
            payload["total_questions"] = breakdown.total_questions
            logger.debug(
                "Included score_full data for submission_id=%s -> %s/%s",
                submission.id,
                breakdown.correct_answers,
                breakdown.total_questions,
            )
        else:
            payload["correct_answers"] = None
            payload["total_questions"] = None
            logger.warning(
                "score_full requested but breakdown unavailable for submission_id=%s",
                submission.id,
            )
    else:
        payload["correct_answers"] = None
        payload["total_questions"] = None

    logger.debug(
        "Final payload for submission_id=%s: %s",
        submission.id,
        {k: payload[k] for k in ("score", "percentile", "position", "correct_answers", "total_questions")},
    )
    payload["answers_review"] = (
        build_answers_review(exam=exam, submission=submission, db=db)
        if getattr(exam, "validated_tribunal", False)
        else None
    )
    return payload
