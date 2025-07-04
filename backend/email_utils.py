import smtplib
from email.message import EmailMessage
import os
from dotenv import load_dotenv

load_dotenv()

SMTP_USER = os.environ["SMTP_USER"]
SMTP_PASS = os.environ["SMTP_PASS"]
SMTP_SERVER = os.environ["SMTP_SERVER"]
SMTP_PORT = int(os.environ["SMTP_PORT"])
ADMIN_EMAIL = os.environ["ADMIN_EMAIL"]

def send_emails(user_email: str, pdf_bytes: bytes, user_name: str, user_surname: str) -> None:
    subject = "Confirmación de inscripción"
    body = f"""
    Hola {user_name} {user_surname},

    Gracias por completar tu inscripción. Adjuntamos el PDF generado con tus datos.

    Un saludo,
    Equipo de OpositaTCAE
    """

    # Crear mensaje base
    def build_msg(to_email: str) -> EmailMessage:
        msg = EmailMessage()
        msg["Subject"] = subject
        msg["From"] = SMTP_USER
        msg["To"] = to_email
        msg.set_content(body, subtype="plain")
        msg.add_attachment(
            pdf_bytes,
            maintype="application",
            subtype="pdf",
            filename="inscripcion.pdf"
        )
        return msg

    msg_user = build_msg(user_email)
    msg_admin = build_msg(ADMIN_EMAIL)

    with smtplib.SMTP(SMTP_SERVER, SMTP_PORT) as smtp:
        smtp.starttls()
        smtp.login(SMTP_USER, SMTP_PASS)
        smtp.send_message(msg_user)
        smtp.send_message(msg_admin)
