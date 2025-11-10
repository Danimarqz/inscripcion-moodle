import asyncio
import os
import time
from collections import OrderedDict
from typing import Iterable, Tuple

from fastapi import Request, status
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.types import ASGIApp, Receive, Scope, Send


def _parse_path_prefixes(raw_value: str | None) -> Tuple[str, ...]:
    if not raw_value:
        return ("/submit-exam",)
    prefixes = tuple(
        prefix.strip()
        for prefix in raw_value.split(",")
        if prefix.strip()
    )
    return prefixes or ("/submit-exam",)


def _resolve_client_ip(request: Request) -> str:
    forwarded_for = request.headers.get("x-forwarded-for")
    if forwarded_for:
        return forwarded_for.split(",")[0].strip()
    real_ip = request.headers.get("x-real-ip")
    if real_ip:
        return real_ip.strip()
    return request.client.host if request.client else "unknown"


class ConstantMemoryRateLimiterMiddleware(BaseHTTPMiddleware):
    """Sliding-window IP limiter with bounded memory usage."""

    def __init__(
        self,
        app: ASGIApp,
        *,
        window_seconds: int | None = None,
        max_requests: int | None = None,
        path_prefixes: Iterable[str] | None = None,
    ) -> None:
        super().__init__(app)
        self.window_seconds = max(
            window_seconds
            if window_seconds is not None
            else int(os.getenv("RATE_LIMIT_SECONDS", "20")),
            1,
        )
        self.max_requests = max(
            max_requests
            if max_requests is not None
            else int(os.getenv("RATE_LIMIT_MAX_REQUESTS", "1")),
            1,
        )
        prefixes = (
            tuple(path_prefixes)
            if path_prefixes is not None
            else _parse_path_prefixes(
                os.getenv("RATE_LIMIT_PATH_PREFIXES")
            )
        )
        self.path_prefixes = tuple(sorted(prefixes))
        self._hits: "OrderedDict[str, tuple[float, int]]" = OrderedDict()
        self._lock = asyncio.Lock()

    def _should_limit(self, path: str) -> bool:
        return any(path.startswith(prefix) for prefix in self.path_prefixes)

    def _evict_expired(self, now: float) -> None:
        while self._hits:
            ip, (reset_at, _) = next(iter(self._hits.items()))
            if reset_at > now:
                break
            self._hits.popitem(last=False)

    async def dispatch(self, request: Request, call_next):
        path = request.url.path or "/"
        if not self._should_limit(path):
            return await call_next(request)

        client_host = _resolve_client_ip(request)
        now = time.monotonic()

        async with self._lock:
            self._evict_expired(now)

            reset_at, count = self._hits.get(
                client_host,
                (now + self.window_seconds, 0),
            )
            if reset_at <= now:
                reset_at = now + self.window_seconds
                count = 0

            if count >= self.max_requests:
                retry_after = max(1, int(reset_at - now))
                return JSONResponse(
                    status_code=status.HTTP_429_TOO_MANY_REQUESTS,
                    content={
                        "detail": "Demasiadas solicitudes, intentalo mas tarde",
                        "retry_after": retry_after,
                    },
                    headers={"Retry-After": str(retry_after)},
                )

            self._hits[client_host] = (reset_at, count + 1)
            self._hits.move_to_end(client_host)

        return await call_next(request)

    async def __call__(self, scope: Scope, receive: Receive, send: Send) -> None:
        await super().__call__(scope, receive, send)
