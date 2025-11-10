import logging

from fastapi import FastAPI, HTTPException, status
from fastapi.middleware.cors import CORSMiddleware
from models.registerdata import RegisterData
from services.register.pdf_utils import generate_pdf
from services.register.email_utils import send_emails
from services.register.moodle_api import create_moodle_user
from services.register.gsheet_api import post_registration_to_gsheet
from rate_limiter import ConstantMemoryRateLimiterMiddleware
from logging_config import configure_logging

configure_logging()
logger = logging.getLogger(__name__)

register_app = FastAPI()
register_app.add_middleware(
    ConstantMemoryRateLimiterMiddleware,
    path_prefixes=("/",),
)

register_app.add_middleware(
    CORSMiddleware,
    allow_origins=["https://inscripcion.opositatcae.es"],
    allow_methods=["OPTIONS", "POST"],
    allow_headers=["*"],
)

@register_app.post("/")
async def register(data: RegisterData):
    if hasattr(data, "website") and data.website:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Spam detectado")
    error_creando_moodle = False
    try:
        form_data = data.model_dump()
        pdf_bytes = generate_pdf(form_data)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error al generar el PDF: {str(e)}"
        )
    try:
        create_moodle_user(form_data)
    except Exception:
        logger.exception("Error al crear usuario en Moodle")
        error_creando_moodle = True

    try:
        send_emails(data.email, pdf_bytes, data.name, data.surname, error_creando_moodle)
    except Exception as exc:
        logger.exception("Error en envio de emails")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error al enviar el correo: {str(exc)}",
        )

    try:
        post_registration_to_gsheet(form_data)
    except Exception as exc:
        logger.exception("Error en envio a gsheet")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error al registrar en Google Sheets: {str(exc)}",
        )

    if error_creando_moodle:
        logger.info("Se envia el mensaje modificado por error creando usuario en Moodle")
        return {"message": "Tu inscripción se gestionará lo antes posible"}
    return {"message": "Inscripción completada correctamente"}
