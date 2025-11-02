import logging
import os
from typing import Optional


def _resolve_level(level_name: str) -> int:
    sanitized = (level_name or "").strip().upper()
    return getattr(logging, sanitized, logging.INFO)


def configure_logging(level: Optional[str] = None) -> None:
    """Configure root logging once with timestamped format."""
    desired_level = level or os.getenv("LOG_LEVEL", "INFO")
    resolved_level = _resolve_level(desired_level)

    root_logger = logging.getLogger()
    if not root_logger.handlers:
        logging.basicConfig(
            level=resolved_level,
            format="%(asctime)s %(levelname)s [%(name)s] %(message)s",
        )
    else:
        root_logger.setLevel(resolved_level)
