from sqlalchemy.orm import Session
from db.models import Exam, Question, UserExamSubmission, UserAnswer
from models.exam import ExamSubmission
from typing import List

def calculate_percentile(user_score: float, all_scores: List[float]) -> float:
    if not all_scores:
        return 100.0
    count_less = sum(1 for score in all_scores if score < user_score)
    return round(100 * count_less / len(all_scores), 2)

def process_exam_submission(data: ExamSubmission, db: Session):
    existing = db.query(UserExamSubmission).filter_by(email=data.email, exam_id=data.exam_id).first()
    if existing:
        raise Exception("Ya has enviado este examen")

    questions = db.query(Question).filter(Question.exam_id == data.exam_id).all()
    questions_dict = {q.id: q for q in questions}
    exam = db.query(Exam).filter(Exam.id == data.exam_id).first()
    correct_count = 0
    for ans in data.answers:
        question = questions_dict.get(ans.question_id)
        if not question:
            raise Exception(f"Pregunta {ans.question_id} no encontrada")
        if ans.answer.upper() == question.correct_option.upper():
            correct_count += 1

    score = correct_count / len(questions) * 100

    submission = UserExamSubmission(
        email=data.email,
        dni=data.dni,
        exam_id=data.exam_id,
        score=score,
        percentile=0
    )
    db.add(submission)
    db.commit()
    db.refresh(submission)

    for ans in data.answers:
        ua = UserAnswer(
            submission_id=submission.id,
            question_id=ans.question_id,
            answer=ans.answer.upper()
        )
        db.add(ua)
    db.commit()

    all_scores = [s.score for s in db.query(UserExamSubmission).filter_by(exam_id=data.exam_id).all()]
    percentile = calculate_percentile(score, all_scores)
    submission.percentile = percentile
    db.commit()
    if exam.show_response:
        return {"score": score, "percentile": percentile, "message": "Examen enviado correctamente"}
    else:
        return {"message": "Examen enviado correctamente"}
