from sqlalchemy import Column, Integer, String, Boolean, ForeignKey, Float, DateTime
from sqlalchemy.orm import declarative_base, relationship
from datetime import datetime, timezone

Base = declarative_base()

class Exam(Base):
    __tablename__ = 'exam'

    id = Column(Integer, primary_key=True)
    name = Column(String(255), nullable=False)
    is_active = Column(Boolean, default=False)
    show_response = Column(Boolean, default=False)
    questions = relationship('Question', back_populates='exam', cascade="all, delete-orphan")
    submissions = relationship('UserExamSubmission', back_populates='exam', cascade="all, delete-orphan")

class Question(Base):
    __tablename__ = 'question'

    id = Column(Integer, primary_key=True)
    exam_id = Column(Integer, ForeignKey('exam.id'), nullable=False)
    correct_option = Column(String(1), nullable=False)  # 'A', 'B', 'C', 'D'

    exam = relationship('Exam', back_populates='questions')

class UserExamSubmission(Base):
    __tablename__ = 'user_exam_submission'

    id = Column(Integer, primary_key=True)
    email = Column(String(255), nullable=False)
    dni = Column(String(20), nullable=False)
    name = Column(String(100), nullable=False)
    surname = Column(String(255), nullable=False)
    exam_id = Column(Integer, ForeignKey('exam.id'), nullable=False)
    score = Column(Float, nullable=True)       # nº de aciertos o porcentaje
    percentile = Column(Float, nullable=True)  # percentil sobre otros usuarios
    submitted_at = Column(DateTime, default=datetime.now(timezone.utc))

    exam = relationship('Exam', back_populates='submissions')
    answers = relationship('UserAnswer', back_populates='submission')

class UserAnswer(Base):
    __tablename__ = 'user_answer'

    id = Column(Integer, primary_key=True)
    submission_id = Column(Integer, ForeignKey('user_exam_submission.id'), nullable=False)
    question_id = Column(Integer, ForeignKey('question.id'), nullable=False)
    answer = Column(String(1), nullable=False)  # 'A', 'B', 'C', 'D'

    submission = relationship('UserExamSubmission', back_populates='answers')

class AdminUser(Base):
    __tablename__ = 'admin_user'

    id = Column(Integer, primary_key=True)
    username = Column(String(150), unique=True, nullable=False)
    password_hash = Column(String(255), nullable=False)