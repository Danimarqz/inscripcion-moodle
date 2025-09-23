import time
import typing as t

from fastapi import HTTPException, Request, status


_RATE_LIMIT_SECONDS = 20  # tiempo mínimo entre envíos
_CLEANUP_MAX_ENTRIES = 10_000  # límite de tamaño antes de compactar

_ip_hits: dict[str, float] = {}
_last_cleanup = 0.0


def _compact(now: float) -> None:
    """Elimina entradas expiradas cuando crece demasiado el diccionario."""
    global _last_cleanup

    if len(_ip_hits) < _CLEANUP_MAX_ENTRIES:
        return

    expired = [ip for ip, ts in _ip_hits.items() if now - ts > _RATE_LIMIT_SECONDS]
    for ip in expired:
        _ip_hits.pop(ip, None)

    _last_cleanup = now


def check_rate_limit(request: Request) -> None:
    """Permite una petición por IP cada _RATE_LIMIT_SECONDS segundos."""
    global _last_cleanup

    now = time.time()
    ip = request.client.host if request.client else "unknown"

    last = _ip_hits.get(ip)
    if last is not None and now - last < _RATE_LIMIT_SECONDS:
        raise HTTPException(
            status_code=status.HTTP_429_TOO_MANY_REQUESTS,
            detail="Demasiadas solicitudes, inténtalo más tarde",
        )

    _ip_hits[ip] = now

    if now - _last_cleanup > _RATE_LIMIT_SECONDS:
        _compact(now)
