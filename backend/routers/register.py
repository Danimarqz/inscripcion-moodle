from fastapi import FastAPI, Request, HTTPException, status
from fastapi.middleware.cors import CORSMiddleware
from models.registerdata import RegisterData
from services.register.pdf_utils import generate_pdf
from services.register.email_utils import send_emails
from services.register.moodle_api import create_moodle_user
from rate_limiter import check_rate_limit

register_app = FastAPI()

register_app.add_middleware(
    CORSMiddleware,
    allow_origins=["https://inscripcion.opositatcae.es"],
    allow_methods=["OPTIONS", "POST"],
    allow_headers=["*"],
)

@register_app.post("/register")
async def register(request: Request, data: RegisterData):
    check_rate_limit(request)
    if hasattr(data, "website") and data.website:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Spam detectado")

    try:
        form_data = data.model_dump()
        pdf_bytes = generate_pdf(form_data)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error al generar el PDF: {str(e)}")

    try:
        send_emails(data.email, pdf_bytes, data.name, data.surname)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error al enviar el correo: {str(e)}")

    try:
        create_moodle_user(form_data)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error al crear usuario en Moodle: {str(e)}")

    return {"message": "Inscripción completada correctamente"}