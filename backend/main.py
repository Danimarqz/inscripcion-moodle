import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from db.models import Base
from db.database import engine
from logging_config import configure_logging
from routers import admin, public, register

configure_logging()
logger = logging.getLogger(__name__)

@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("Iniciando app...")
    Base.metadata.create_all(bind=engine)
    yield
    logger.info("Apagando app...")

app = FastAPI(lifespan=lifespan)

# TODO change on prod.

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(public.router)
app.include_router(admin.router, prefix="/admin")

# CORS Diferente
app.mount("/register", register.register_app)