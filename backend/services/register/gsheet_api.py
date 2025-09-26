import re
from datetime import datetime
from typing import Any, Dict

import requests

from config import API_GSHEET

REQUEST_TIMEOUT = 10

COURSE_LABELS: Dict[str, str] = {
    "ingesa": "INGESA (CEUTA Y MELILLA)",
    "sergas": "SERGAS (GALICIA)",
    "gva": "COMUNIDAD VALENCIANA (GVA)",
    "aragon": "SERVICIO ARAGONES DE SALUD",
    "sescam": "SESCAM (CASTILLA LA MANCHA)",
    "osakidetza": "OSAKIDETZA (PAIS VASCO)",
    "sacyl": "SACYL (CASTILLA Y LEON)",
    "osasunbidea": "OSASUNBIDEA (NAVARRA)",
    "seris": "SERIS (LA RIOJA)",
    "cantabria": "SCS (CANTABRIA)",
    "canarias": "SCS (CANARIAS)",
    "sms": "SMS (MURCIA)",
    "sespa": "SESPA (ASTURIAS)",
    "imserso": "IMSERSO",
    "ses": "SES (EXTREMADURA)",
    "era": "ERA (ASTURIAS) - CONSULTAR",
    "ib-salut": "IB-SALUT (ISLAS BALEARES)",
    "xunta": "XUNTA DE GALICIA",
    "sas": "SAS (ANDALUCIA)",
    "sermas": "SERMAS (MADRID)",
}


def post_registration_to_gsheet(form_data: Dict[str, Any]) -> None:
    if not API_GSHEET:
        raise RuntimeError("API_GSHEET no esta configurado")

    payload = _build_payload(form_data)
    response = requests.post(API_GSHEET, json=payload, timeout=REQUEST_TIMEOUT)

    if response.status_code != 200:
        raise RuntimeError(f"Google Sheets respondio con codigo {response.status_code}: {response.text}")

    try:
        body = response.json()
    except ValueError as exc:
        raise RuntimeError(f"Respuesta no valida desde Google Sheets: {response.text}") from exc

    if body.get("status") != "ok":
        message = body.get("message", "Respuesta inesperada del script de Google Sheets")
        raise RuntimeError(f"Google Sheets devolvio un error: {message}")


def _build_payload(data: Dict[str, Any]) -> Dict[str, Any]:
    modality = _normalize_string(data.get("modality"))

    return {
        "CURSO": _map_course(data.get("course")),
        "GRUPO": _extract_group(modality),
        "IMPORTE/MES": _extract_amount(modality),
        "PRIMER PAGO": _format_first_payment_month(data.get("startdate")),
        "OBSERVACIONES": _build_observaciones(data),
        "FECHA ALTA": datetime.now().strftime("%d/%m/%Y"),
        "Nombre": _normalize_string(data.get("name")),
        "Apellido": _normalize_string(data.get("surname")),
        "Nacimiento": _format_birth_date(data.get("dob")),
        "DNI": _normalize_string(data.get("dni")),
        "Domicilio": _normalize_string(data.get("address")),
        "Provincia": _normalize_string(data.get("city")),
        "Localidad": _normalize_string(data.get("locality")),
        "CP": _normalize_string(data.get("postalcode")),
        "Pa\u00eds": _normalize_string(data.get("country")),
        "Tel\u00e9fono": _normalize_string(data.get("phone")),
        "Email": _normalize_string(data.get("email")),
        "IBAN": _normalize_string(data.get("iban")),
        "\u00bfC\u00f3mo nos conociste?": _normalize_string(data.get("discover")),
    }


def _map_course(course_key: Any) -> str:
    if not isinstance(course_key, str):
        return ""
    key = course_key.strip().lower()
    return COURSE_LABELS.get(key, course_key)


def _format_birth_date(dob: Any) -> str:
    if not isinstance(dob, str):
        return ""
    try:
        return datetime.strptime(dob, "%Y-%m-%d").strftime("%d/%m/%Y")
    except ValueError:
        return dob


def _format_first_payment_month(start: Any) -> str:
    if not isinstance(start, str):
        return ""
    try:
        return datetime.strptime(start, "%Y-%m").strftime("%m/%Y")
    except ValueError:
        return start


def _extract_group(modality: str) -> str:
    if not modality:
        return ""
    match = re.search(r"\(([^)]+)\)", modality)
    if match:
        return match.group(1).strip()
    return modality


def _extract_amount(modality: str) -> str:
    if not modality:
        return ""
    match = re.search(r"(\d+[\.,]?\d*)", modality)
    if not match:
        return modality
    amount = match.group(1).replace(",", ".")
    return f"{amount} EUR/mes"


def _build_observaciones(data: Dict[str, Any]) -> str:
    payment = _normalize_string(data.get("payment"))
    if payment:
        return f"Forma de pago: {payment}"
    return ""


def _normalize_string(value: Any) -> str:
    if value is None:
        return ""
    return str(value).strip()
