import hashlib
import os
import time
from threading import RLock
from typing import Any


class TTLCache:
    def __init__(self, default_ttl: int = 30):
        self.default_ttl = max(default_ttl, 1)
        self._store: dict[str, tuple[float, Any]] = {}
        self._lock = RLock()

    def get(self, key: str) -> Any | None:
        with self._lock:
            item = self._store.get(key)
            if not item:
                return None
            expires_at, value = item
            if expires_at <= time.time():
                self._store.pop(key, None)
                return None
            return value

    def set(self, key: str, value: Any, ttl: int | None = None) -> None:
        lifetime = ttl if ttl is not None else self.default_ttl
        expires_at = time.time() + max(lifetime, 1)
        with self._lock:
            self._store[key] = (expires_at, value)

    def invalidate(self, key: str) -> None:
        with self._lock:
            self._store.pop(key, None)

    def invalidate_prefix(self, prefix: str) -> None:
        with self._lock:
            keys_to_remove = [key for key in self._store if key.startswith(prefix)]
            for key in keys_to_remove:
                self._store.pop(key, None)

    def clear(self) -> None:
        with self._lock:
            self._store.clear()


PUBLIC_CACHE_TTL = int(os.getenv("PUBLIC_CACHE_TTL", "30"))
public_cache = TTLCache(default_ttl=PUBLIC_CACHE_TTL)
EXAMS_CACHE_KEY = "public:exams"


def questions_cache_key(exam_id: int) -> str:
    return f"public:questions:{exam_id}"


def submission_check_cache_key(exam_id: int, email: str, dni: str) -> str:
    normalized_email = email.strip().lower()
    normalized_dni = dni.strip().upper()
    fingerprint = f"{exam_id}:{normalized_email}:{normalized_dni}"
    digest = hashlib.sha256(fingerprint.encode("utf-8")).hexdigest()
    return f"public:check:{exam_id}:{digest}"


def invalidate_check_cache_for_exam(exam_id: int) -> None:
    public_cache.invalidate_prefix(f"public:check:{exam_id}:")


def invalidate_exam_cache(exam_id: int | None = None) -> None:
    public_cache.invalidate(EXAMS_CACHE_KEY)
    if exam_id is not None:
        public_cache.invalidate(questions_cache_key(exam_id))
        invalidate_check_cache_for_exam(exam_id)


def invalidate_all_questions_cache() -> None:
    public_cache.invalidate_prefix("public:questions:")
