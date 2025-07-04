from fastapi import FastAPI, status, Request, HTTPException
from fastapi.middleware.cors import CORSMiddleware

from moodle_api import create_moodle_user
from rate_limiter import check_rate_limit
from registerdata import RegisterData
from pdf_utils import generate_pdf
from email_utils import send_emails

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["https://inscripcion.opositatcae.es"], 
    allow_methods=["OPTIONS", "POST"],
    allow_headers=["*"],
)

@app.post("/register")
async def register(request: Request, data: RegisterData):
    check_rate_limit(request)

    if hasattr(data, "website") and data.website:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Spam detectado"
        )

    try:
        form_data = data.model_dump()
        pdf_bytes = generate_pdf(form_data)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error al generar el PDF: {str(e)}"
        )

    try:
        send_emails(data.email, pdf_bytes, data.name, data.surname)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error al enviar el correo: {str(e)}"
        )
    try:
        moodle_user_id = create_moodle_user(form_data)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error al crear usuario en Moodle: {str(e)}"
        )
    return {"message": "Inscripción completada correctamente"}