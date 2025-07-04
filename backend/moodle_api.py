import requests
import os
from dotenv import load_dotenv

load_dotenv()

MOODLE_URL = os.environ["MOODLE_URL"]
MOODLE_TOKEN = os.environ["MOODLE_TOKEN"]

def create_moodle_user(data: dict):
    username = data["email"]

    # Campos personalizados base
    customfields = [
        {"type": "dni", "value": data["dni"]},
        {"type": "conocer", "value": data["discover"]}
    ]

    # Activar el campo extra según el curso seleccionado
    valid_courses = {
        "ingesa", "sergas", "gva", "aragon", "sescam", "osakidetza", "sacyl",
        "osasunbidea", "seris", "cantabria", "canarias", "sms", "sespa",
        "imserso", "ses", "era"
    }

    selected_course = data["course"]
    if selected_course in valid_courses:
        customfields.append({"type": selected_course, "value": "true"})

    payload = {
        "wstoken": MOODLE_TOKEN,
        "wsfunction": "core_user_create_users",
        "moodlewsrestformat": "json",
        "users[0][username]": username,
        "users[0][firstname]": data["name"],
        "users[0][lastname]": data["surname"],
        "users[0][email]": data["email"],
        "users[0][auth]": "manual",
        "users[0][password]": generate_password(data["dni"])
    }

    # Añadir campos personalizados al payload
    for i, field in enumerate(customfields):
        payload[f"users[0][customfields][{i}][type]"] = field["type"]
        payload[f"users[0][customfields][{i}][value]"] = field["value"]

    response = requests.post(f"{MOODLE_URL}/webservice/rest/server.php", data=payload)

    if response.status_code != 200:
        raise Exception(f"Error al conectar con Moodle: {response.text}")

    result = response.json()

    if isinstance(result, dict) and result.get("exception"):
        raise Exception(f"Error de Moodle: {result['message']}")

    return result[0]["id"]

def generate_password(dni: str) -> str:
    return dni.upper()
