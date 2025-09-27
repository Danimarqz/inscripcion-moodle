import smtplib
from email.message import EmailMessage
from config import SMTP_USER, SMTP_PASS, SMTP_SERVER, SMTP_PORT, ADMIN_EMAIL

def send_emails(user_email: str, pdf_bytes: bytes, user_name: str, user_surname: str, error_creando_moodle: bool) -> None:
    subject = "Confirmación de inscripción"
    error_moodle = f"\nEl email {user_email} ya estaba usado en moodle. Por favor, revísalo.\n"
    body_admin = f"""
    Hola {user_name} {user_surname},
    {error_moodle if error_creando_moodle else ""}
    Gracias por completar tu inscripción. Adjuntamos el PDF generado con tus datos.

    Un saludo,
    Equipo de OpositaTCAE
    """
    body_user = f"""
    Hola {user_name} {user_surname},

    Gracias por completar tu inscripción. Adjuntamos el PDF generado con tus datos.

    Un saludo,
    Equipo de OpositaTCAE
    """
    # Crear mensaje base
    def build_msg(to_email: str, body: str) -> EmailMessage:
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

    msg_user = build_msg(user_email, body_user)
    msg_admin = build_msg(ADMIN_EMAIL, body_admin)

    with smtplib.SMTP(SMTP_SERVER, SMTP_PORT) as smtp:
        smtp.starttls()
        smtp.login(SMTP_USER, SMTP_PASS)
        smtp.send_message(msg_user)
        smtp.send_message(msg_admin)
