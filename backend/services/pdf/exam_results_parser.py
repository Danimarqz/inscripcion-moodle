from __future__ import annotations
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, List, Sequence
import pandas as pd
from tabula import read_pdf as tabula_read_pdf


HEADER_KEYWORDS = ("DNI", "APELLIDO", "NOMBRE")
FIRST_COLUMN_VALID_PATTERN = re.compile(r"[0-9#]")
DEFAULT_ENCODING = "latin-1"


@dataclass(slots=True)
class ParseSummary:
    raw_tables: List[pd.DataFrame]
    useful_tables: List[pd.DataFrame]
    combined: pd.DataFrame


def _clean_cell(value: object) -> str:
    if value is None:
        return ""
    text = str(value).replace("\xa0", " ").replace("\r", " ").replace("\n", " ")
    text = text.strip()
    return "" if text.lower() in {"nan", "none"} else text


def _row_is_header_like(values: Sequence[str]) -> bool:
    tokens = [str(value).strip().upper() for value in values if str(value).strip()]
    if not tokens:
        return False
    joined = " ".join(tokens)
    return any(keyword in joined for keyword in HEADER_KEYWORDS)


def _normalize_table(df: pd.DataFrame) -> pd.DataFrame:
    if df is None or df.empty:
        return pd.DataFrame()

    working = df.map(_clean_cell).copy()
    working = working.loc[:, ~(working == "").all(axis=0)]

    if working.empty:
        return pd.DataFrame()

    header_mask = working.apply(lambda row: _row_is_header_like(row.values), axis=1)
    working = working.loc[~header_mask]
    working = working.loc[
        ~(
            working.apply(
                lambda row: all(str(val) == "" for val in row.values), axis=1
            )
        )
    ]

    if working.empty:
        return pd.DataFrame()

    first_col = working.iloc[:, 0].astype(str)
    mask_non_empty = first_col.str.strip() != ""
    mask_valid = first_col.str.contains(FIRST_COLUMN_VALID_PATTERN, na=False)
    working = working.loc[mask_non_empty & mask_valid]

    if working.empty:
        return pd.DataFrame()

    records: list[dict[str, str]] = []

    for row in working.itertuples(index=False, name=None):
        row_values = [str(value).strip() for value in row]
        dni = row_values[0]
        if not dni:
            continue

        rest = [value for value in row_values[1:] if value]
        if not rest:
            continue

        apellido_1 = rest[0]
        apellido_2 = ""
        nombre_tokens: list[str] = []

        if len(rest) >= 3:
            apellido_2 = rest[1]
            nombre_tokens = rest[2:]
        elif len(rest) == 2:
            nombre_tokens = rest[1:]

        nombre = " ".join(nombre_tokens).strip()

        records.append(
            {
                "dni": dni,
                "apellido_1": apellido_1,
                "apellido_2": apellido_2,
                "nombre": nombre,
            }
        )

    if not records:
        return pd.DataFrame()

    normalized = pd.DataFrame.from_records(records)
    return normalized[["dni", "apellido_1", "apellido_2", "nombre"]]


def _extract_useful_tables(tables: Iterable[pd.DataFrame]) -> List[pd.DataFrame]:
    useful: list[pd.DataFrame] = []
    for raw_df in tables:
        normalized = _normalize_table(raw_df)
        if not normalized.empty:
            useful.append(normalized)
    return useful


def _combine_tables(tables: Iterable[pd.DataFrame]) -> pd.DataFrame:
    tables_list = list(tables)
    if not tables_list:
        return pd.DataFrame(columns=["dni", "apellido_1", "apellido_2", "nombre"])
    return pd.concat(tables_list, ignore_index=True)


def _read_pdf_tables(
    pdf_path: Path,
    *,
    pages: str = "all",
    lattice: bool = False,
    encoding: str = DEFAULT_ENCODING,
) -> list[pd.DataFrame]:
    resolved = Path(pdf_path)
    if not resolved.exists():
        raise FileNotFoundError(f"Cannot find PDF at {resolved}")

    dataframes = tabula_read_pdf(
        str(resolved),
        pages=pages,
        lattice=lattice,
        guess=False,
        encoding=encoding,
        pandas_options={"dtype": str, "header": None},
    )

    if not dataframes:
        return []

    return [df.dropna(how="all").fillna("") for df in dataframes]


def parse_exam_results(
    pdf_path: Path,
    *,
    pages: str = "all",
    lattice: bool = False,
    encoding: str = DEFAULT_ENCODING,
) -> ParseSummary:
    raw_tables = _read_pdf_tables(
        pdf_path,
        pages=pages,
        lattice=lattice,
        encoding=encoding,
    )
    useful_tables = _extract_useful_tables(raw_tables)
    combined = _combine_tables(useful_tables)
    return ParseSummary(
        raw_tables=raw_tables,
        useful_tables=useful_tables,
        combined=combined,
    )


def smart_parse_exam_results(
    pdf_path: Path,
    *,
    pages: str = "all",
    lattice: bool = False,
    encoding: str = DEFAULT_ENCODING,
    min_rows_for_guess: int = 200,
) -> ParseSummary:
    first_pass = parse_exam_results(
        pdf_path,
        pages=pages,
        lattice=lattice,
        encoding=encoding,
    )

    if len(first_pass.combined) >= min_rows_for_guess:
        return first_pass

    second_pass = parse_exam_results(
        pdf_path,
        pages=pages,
        lattice=lattice,
        encoding=encoding,
    )

    if len(second_pass.combined) > len(first_pass.combined):
        return ParseSummary(
            raw_tables=second_pass.raw_tables,
            useful_tables=second_pass.useful_tables,
            combined=second_pass.combined,
        )

    return first_pass
