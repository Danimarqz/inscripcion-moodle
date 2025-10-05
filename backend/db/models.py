from datetime import datetime, timezone

from sqlalchemy import (
    Boolean,
    Column,
    DateTime,
    Float,
    ForeignKey,
    Integer,
    String,
    UniqueConstraint,
)
from sqlalchemy.orm import declarative_base, relationship

Base = declarative_base()


class ExamUser(Base):
    __tablename__ = "exam_user"

    id = Column(Integer, primary_key=True)
    name = Column(String(150), nullable=False)
    surname = Column(String(255), nullable=False)
    email = Column(String(255), nullable=True)
    dni = Column(String(20), nullable=False, unique=True)
    created_at = Column(DateTime, default=datetime.now(timezone.utc))

    submissions = relationship(
        "UserExamSubmission",
        back_populates="user",
        cascade="all, delete-orphan",
    )
    official_results = relationship(
        "ExamOfficialResult",
        back_populates="user",
        cascade="all, delete-orphan",
    )


class Exam(Base):
    __tablename__ = "exam"

    id = Column(Integer, primary_key=True)
    name = Column(String(255), nullable=False)
    is_active = Column(Boolean, default=False)
    show_response = Column(Boolean, default=False)
    questions = relationship(
        "Question",
        back_populates="exam",
        cascade="all, delete-orphan",
    )
    submissions = relationship(
        "UserExamSubmission",
        back_populates="exam",
        cascade="all, delete-orphan",
    )
    official_results = relationship(
        "ExamOfficialResult",
        back_populates="exam",
        cascade="all, delete-orphan",
    )


class Question(Base):
    __tablename__ = "question"

    id = Column(Integer, primary_key=True)
    exam_id = Column(Integer, ForeignKey("exam.id"), nullable=False)
    correct_option = Column(String(1), nullable=False)  # 'A', 'B', 'C', 'D'
    is_active = Column(Boolean, nullable=False, default=True)

    exam = relationship("Exam", back_populates="questions")


class UserExamSubmission(Base):
    __tablename__ = "user_exam_submission"

    id = Column(Integer, primary_key=True)
    user_id = Column(Integer, ForeignKey("exam_user.id"), nullable=False)
    exam_id = Column(Integer, ForeignKey("exam.id"), nullable=False)
    score = Column(Float, nullable=True)  # numero de aciertos o porcentaje
    percentile = Column(Float, nullable=True)  # percentil sobre otros usuarios
    submitted_at = Column(DateTime, default=datetime.now(timezone.utc))

    exam = relationship("Exam", back_populates="submissions")
    user = relationship("ExamUser", back_populates="submissions")
    answers = relationship(
        "UserAnswer",
        back_populates="submission",
        cascade="all, delete-orphan",
    )


class UserAnswer(Base):
    __tablename__ = "user_answer"

    id = Column(Integer, primary_key=True)
    submission_id = Column(
        Integer, ForeignKey("user_exam_submission.id"), nullable=False
    )
    question_id = Column(Integer, ForeignKey("question.id"), nullable=False)
    answer = Column(String(1), nullable=False)  # 'A', 'B', 'C', 'D'

    submission = relationship("UserExamSubmission", back_populates="answers")


class ExamOfficialResult(Base):
    __tablename__ = "exam_official_result"
    __table_args__ = (
        UniqueConstraint("exam_id", "user_id", name="uq_exam_result_exam_user"),
    )

    id = Column(Integer, primary_key=True)
    exam_id = Column(Integer, ForeignKey("exam.id"), nullable=False)
    user_id = Column(Integer, ForeignKey("exam_user.id"), nullable=True)
    dni_masked = Column(String(20), nullable=False)
    apellido_1 = Column(String(255), nullable=False)
    apellido_2 = Column(String(255), nullable=True)
    nombre = Column(String(255), nullable=False)
    created_at = Column(DateTime, default=datetime.now(timezone.utc))

    exam = relationship("Exam", back_populates="official_results")
    user = relationship("ExamUser", back_populates="official_results")


class AdminUser(Base):
    __tablename__ = "admin_user"

    id = Column(Integer, primary_key=True)
    username = Column(String(150), unique=True, nullable=False)
    password_hash = Column(String(255), nullable=False)

