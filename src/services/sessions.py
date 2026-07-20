import hashlib
import secrets
import threading
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta

from models import User

ACCESS_TOKEN_LIFETIME = timedelta(minutes=30)
REFRESH_IDLE_LIFETIME = timedelta(days=90)


@dataclass(frozen=True)
class Session:
    user: User
    family_id: bytes
    expires_at: datetime


class SessionStore:
    def __init__(self) -> None:
        self._sessions: dict[bytes, Session] = {}
        self._lock = threading.Lock()

    def create(self, user: User, family_id: bytes) -> str:
        token = generate_token()
        now = _now()
        with self._lock:
            self._sessions = {
                key: value for key, value in self._sessions.items() if value.expires_at > now
            }
            self._sessions[digest_token(token)] = Session(
                user=user, family_id=family_id, expires_at=now + ACCESS_TOKEN_LIFETIME
            )
        return token

    def get(self, token: str) -> Session | None:
        key = digest_token(token)
        with self._lock:
            session = self._sessions.get(key)
            if session is None:
                return None
            if session.expires_at <= _now():
                del self._sessions[key]
                return None
            return session

    def revoke_family(self, family_id: bytes) -> None:
        with self._lock:
            self._sessions = {
                key: value
                for key, value in self._sessions.items()
                if not secrets.compare_digest(value.family_id, family_id)
            }

    def revoke_user(self, user_id: int) -> None:
        with self._lock:
            self._sessions = {
                key: value for key, value in self._sessions.items() if value.user.id != user_id
            }


def generate_token() -> str:
    return secrets.token_urlsafe(32)


def digest_token(token: str) -> bytes:
    return hashlib.sha256(token.encode()).digest()


def _now() -> datetime:
    return datetime.now(UTC)
