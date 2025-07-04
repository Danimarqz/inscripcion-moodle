import time
from fastapi import Request, HTTPException, status

ip_request_times = {}
RATE_LIMIT_SECONDS = 60
CLEANUP_INTERVAL = 300  # Cada 5 minutos
_last_cleanup = 0

def check_rate_limit(request: Request):
    global _last_cleanup
    now = time.time()
    ip = request.client.host

    # Limpiar IPs que ya han caducado (cada X segundos)
    if now - _last_cleanup > CLEANUP_INTERVAL:
        expired = [
            ip for ip, ts in ip_request_times.items()
            if now - ts > RATE_LIMIT_SECONDS
        ]
        for ip_to_remove in expired:
            del ip_request_times[ip_to_remove]
        _last_cleanup = now

    # Revisar si esa IP ha hecho petición reciente
    last = ip_request_times.get(ip)
    if last and now - last < RATE_LIMIT_SECONDS:
        raise HTTPException(
            status_code=status.HTTP_429_TOO_MANY_REQUESTS,
            detail="Demasiadas solicitudes, inténtalo más tarde"
        )

    ip_request_times[ip] = now
