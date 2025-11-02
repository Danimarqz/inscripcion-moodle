from __future__ import annotations

import logging
from dataclasses import dataclass
from pathlib import Path
from typing import Optional

from sqlalchemy import func
from sqlalchemy.orm import Session

from db.models import Exam, ExamOfficialResult, ExamUser
from logging_config import configure_logging
from services.exam.submit_exam import normalize_dni
from services.pdf.exam_results_parser import DEFAULT_ENCODING, parse_exam_results

configure_logging()
logger = logging.getLogger(__name__)


@dataclass(slots=True)
class ImportStats:
    exam_id: int
    total_rows: int
    imported_results: int
    created_users: int
    updated_users: int


class ExamResultImportError(RuntimeError):
    """Raised when an official exam result import fails."""


def _compose_surname(apellido_1: str, apellido_2: Optional[str]) -> str:
    parts = [part.strip() for part in (apellido_1, apellido_2) if part and part.strip()]
    return " ".join(parts)


def _normalize_upper(value: Optional[str]) -> str:
    return value.strip().upper() if value else ""


def _mask_to_like_pattern(mask: str) -> tuple[str, bool]:
    if not mask:
        return mask, False

    escaped = mask.replace("\\", "\\\\")
    has_wildcards = "#" in escaped
    escaped = escaped.replace("%", r"\%").replace("_", r"\_")
    pattern = escaped.replace("#", "_")
    return pattern, has_wildcards


def _find_matching_candidate(
    db: Session,
    dni_masked: str,
    name_upper: str,
    surname_upper: str,
    nombre_original: str,
    apellido_original: str,
) -> Optional[ExamUser]:
    normalized_mask = normalize_dni(dni_masked)

    if normalized_mask and "#" not in normalized_mask:
        direct = (
            db.query(ExamUser)
            .filter(ExamUser.dni == normalized_mask)
            .one_or_none()
        )
        if direct:
            return direct

    pattern, has_wildcards = _mask_to_like_pattern(normalized_mask)
    if has_wildcards:
        query = db.query(ExamUser).filter(
            ExamUser.dni.like(pattern, escape="\\")
        )
        if surname_upper:
            query = query.filter(func.upper(ExamUser.surname) == surname_upper)
        if name_upper:
            query = query.filter(func.upper(ExamUser.name) == name_upper)

        matches = query.all()
        if len(matches) == 1:
            return matches[0]
        if len(matches) > 1:
            raise ExamResultImportError(
                f"Coincidencia ambigua para el DNI enmascarado {dni_masked}"
            )

    if name_upper and surname_upper:
        name_query = (
            db.query(ExamUser)
            .filter(func.upper(ExamUser.name) == name_upper)
            .filter(func.upper(ExamUser.surname) == surname_upper)
        )
        matches = name_query.all()
        if len(matches) == 1:
            return matches[0]
        if len(matches) > 1:
            raise ExamResultImportError(
                f"Coincidencia ambigua por nombre y apellidos ({nombre_original} {apellido_original})"
            )

    return None


def import_official_results_from_pdf(
    *,
    db: Session,
    exam_id: int,
    pdf_path: Path,
    encoding: str = DEFAULT_ENCODING,
    lattice: bool = False,
    replace_existing: bool = True,
) -> ImportStats:
    exam = db.query(Exam).filter(Exam.id == exam_id).first()
    if not exam:
        raise ExamResultImportError(f"Examen con id {exam_id} no existe")

    summary = parse_exam_results(
        pdf_path,
        encoding=encoding,
        lattice=lattice,
        guess=False,
    )

    if summary.combined.empty:
        raise ExamResultImportError("El PDF no contiene filas de resultados reconocibles")

    if replace_existing:
        db.query(ExamOfficialResult).filter(ExamOfficialResult.exam_id == exam_id).delete()

    created_users = 0
    updated_users = 0
    imported_results = 0

    for _, row in summary.combined.iterrows():  # type: ignore[attr-defined]
        dni_masked = str(row.get("dni", "")).strip()
        apellido_1 = str(row.get("apellido_1", "")).strip()
        apellido_2_raw = str(row.get("apellido_2", "")).strip() or None
        nombre = str(row.get("nombre", "")).strip()

        if not dni_masked or not apellido_1 or not nombre:
            logger.debug("Skipping row with missing mandatory data: %s", row)
            continue

        surname = _compose_surname(apellido_1, apellido_2_raw)
        surname_for_storage = surname or apellido_1
        name_upper = _normalize_upper(nombre)
        surname_upper = _normalize_upper(surname_for_storage)

        candidate = _find_matching_candidate(
            db,
            dni_masked,
            name_upper,
            surname_upper,
            nombre,
            surname_for_storage,
        )

        if candidate is not None:
            updated_fields = False
            if nombre and _normalize_upper(candidate.name) != name_upper:
                candidate.name = nombre
                updated_fields = True
            if surname_for_storage and _normalize_upper(candidate.surname) != surname_upper:
                candidate.surname = surname_for_storage
                updated_fields = True
            if updated_fields:
                updated_users += 1

        result = ExamOfficialResult(
            exam_id=exam_id,
            user_id=candidate.id if candidate is not None else None,
            dni_masked=dni_masked,
            apellido_1=apellido_1,
            apellido_2=apellido_2_raw,
            nombre=nombre,
        )
        db.add(result)
        imported_results += 1

    db.commit()

    return ImportStats(
        exam_id=exam_id,
        total_rows=len(summary.combined),
        imported_results=imported_results,
        created_users=created_users,
        updated_users=updated_users,
    )
