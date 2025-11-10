from os import getenv

from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

from config import DATABASE_URL

POOL_SIZE = int(getenv("DB_POOL_SIZE", "5"))
MAX_OVERFLOW = int(getenv("DB_MAX_OVERFLOW", "5"))
POOL_RECYCLE = int(getenv("DB_POOL_RECYCLE", "300"))
POOL_TIMEOUT = int(getenv("DB_POOL_TIMEOUT", "30"))

engine = create_engine(
    DATABASE_URL,
    pool_size=max(1, POOL_SIZE),
    max_overflow=max(0, MAX_OVERFLOW),
    pool_recycle=max(60, POOL_RECYCLE),
    pool_timeout=POOL_TIMEOUT,
    echo=False,
    pool_pre_ping=True,
)

SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
