from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from db.models import Base
from db.database import engine
from routers import public, admin
from routers.register import register_app

@asynccontextmanager
async def lifespan(app: FastAPI):
    print("Iniciando app...")
    Base.metadata.create_all(bind=engine)
    yield
    print("Apagando app...")

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
app.mount("/register", register_app)