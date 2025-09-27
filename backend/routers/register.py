from fastapi import FastAPI, Request, HTTPException, status
from fastapi.middleware.cors import CORSMiddleware
from models.registerdata import RegisterData
from services.register.pdf_utils import generate_pdf
from services.register.email_utils import send_emails
from services.register.moodle_api import create_moodle_user
from services.register.gsheet_api import post_registration_to_gsheet
from rate_limiter import check_rate_limit

register_app = FastAPI()

register_app.add_middleware(
    CORSMiddleware,
    allow_origins=["https://inscripcion.opositatcae.es"],
    allow_methods=["OPTIONS", "POST"],
    allow_headers=["*"],
)

@register_app.post("/")
async def register(request: Request, data: RegisterData):
    check_rate_limit(request)
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
    except Exception as e:
        print(f"Error al crear usuario en Moodle: {str(e)}")
        error_creando_moodle = True

    try:
        send_emails(data.email, pdf_bytes, data.name, data.surname, error_creando_moodle)
    except Exception as e:
        print(f"Error en envio de emails: {str(e)}")
        raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=f"Error al enviar el correo: {str(e)}")

    try:
        post_registration_to_gsheet(form_data)
    except Exception as e:
        print(f"Error en envio a gsheet: {str(e)}")
        raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=f"Error al registrar en Google Sheets: {str(e)}")

    if error_creando_moodle:
        print("se envia el mensaje modificado")
        return {"message": "Tu inscripción se gestionará lo antes posible"}
    return {"message": "Inscripción completada correctamente"}
