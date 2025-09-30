import requests
from config import MOODLE_URL, MOODLE_TOKEN

MOODLE_ENDPOINT = f"{MOODLE_URL}/webservice/rest/server.php"
STUDENT_ROLE_ID = 5

VALID_COURSES = {
    "ingesa": 24,
    "sergas": 10,
    "gva": 15,
    "aragon": 3,
    "sescam": 13,
    "osakidetza": 23,
    "sacyl": 26,
    "osasunbidea": 36,
    "seris": 46,
    "cantabria": 35,
    "canarias": 25,
    "sms": 72,
    "sespa": 45,
    "imserso": 18,
    "ses": 12,
    "era": 14,
    "ib-salut": 83,
    "xunta": 89,
    "sas": 19,
    "sermas": 20,
}
EXAMENES_OFICIALES_AL = 16
TECNICA_TEST = 11


def create_moodle_user(data: dict):
    username = data["email"].lower().strip()
    raw_selected_course = data.get("course", "").strip()
    course_key = raw_selected_course.lower() if raw_selected_course else ""

    customfields = [
        {"type": "dni", "value": data["dni"]},
        {"type": "conocer", "value": data["discover"]},
    ]

    if course_key in VALID_COURSES:
        customfields.append({"type": course_key, "value": "true"})

    payload = {
        "users[0][username]": username,
        "users[0][firstname]": data["name"],
        "users[0][lastname]": data["surname"],
        "users[0][email]": data["email"],
        "users[0][auth]": "manual",
        "users[0][password]": generate_password(data["dni"]),
    }

    for i, field in enumerate(customfields):
        payload[f"users[0][customfields][{i}][type]"] = field["type"]
        payload[f"users[0][customfields][{i}][value]"] = field["value"]

    result = call_moodle("core_user_create_users", payload, log_response=True)

    if not isinstance(result, list) or not result or "id" not in result[0]:
        raise Exception("Respuesta inesperada al crear el usuario en Moodle.")

    user_id = result[0]["id"]

    for course_id in resolve_courses_to_enrol(course_key):
        enrol_user_in_course(user_id, course_id)

    return user_id

def call_moodle(function: str, payload: dict, *, log_response: bool = False):
    request_payload = {
        "wstoken": MOODLE_TOKEN,
        "wsfunction": function,
        "moodlewsrestformat": "json",
    }
    request_payload.update(payload)

    response = requests.post(MOODLE_ENDPOINT, data=request_payload)

    if log_response:
        print(response.text)

    if response.status_code != 200:
        raise Exception(f"Error al conectar con Moodle: {response.text}")

    try:
        result = response.json()
    except ValueError as exc:
        raise Exception(f"Respuesta no valida desde Moodle: {response.text}") from exc

    if isinstance(result, dict) and result.get("exception"):
        raise Exception(f"Error de Moodle: {result['message']}")

    return result

def resolve_courses_to_enrol(course_key: str):
    course_ids = []

    if course_key in VALID_COURSES:
        course_ids.append(normalize_course_id(VALID_COURSES[course_key]))

    course_ids.extend([EXAMENES_OFICIALES_AL, TECNICA_TEST])

    # Utiliza dict para mantener el orden y eliminar duplicados.
    return list(dict.fromkeys(course_ids))

def normalize_course_id(course_id):
    try:
        return int(course_id)
    except (TypeError, ValueError) as exc:
        raise Exception(f"ID de curso invalido configurado: {course_id}") from exc

def enrol_user_in_course(user_id: int, course_id: int, role_id: int = STUDENT_ROLE_ID) -> None:
    resolved_course_id = normalize_course_id(course_id)

    call_moodle(
        "enrol_manual_enrol_users",
        {
            "enrolments[0][roleid]": role_id,
            "enrolments[0][userid]": user_id,
            "enrolments[0][courseid]": resolved_course_id,
        },
    )

def generate_password(dni: str) -> str:
    return dni.upper()
