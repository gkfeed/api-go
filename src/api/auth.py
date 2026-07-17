import hashlib
import logging
import secrets
import sqlite3
import threading
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, Request, Response, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from pydantic import BaseModel, ConfigDict

from database import Database
from database.refresh_tokens import InvalidRefreshTokenError, RefreshTokenReuseError
from models import User

from .passwords import DUMMY_PASSWORD_HASH, verify_password

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/auth", tags=["auth"])
bearer_auth = HTTPBearer(auto_error=False)
ACCESS_TOKEN_LIFETIME = timedelta(minutes=30)
REFRESH_IDLE_LIFETIME = timedelta(days=90)


class LoginRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")
    username: str
    password: str


class RefreshRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")
    refresh_token: str


class TokenResponse(BaseModel):
    access_token: str
    refresh_token: str
    token_type: str = "Bearer"
    expires_in: int = int(ACCESS_TOKEN_LIFETIME.total_seconds())
    refresh_expires_in: int = int(REFRESH_IDLE_LIFETIME.total_seconds())


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
        token = _new_token()
        now = _now()
        with self._lock:
            self._sessions = {
                key: value for key, value in self._sessions.items() if value.expires_at > now
            }
            self._sessions[_digest(token)] = Session(
                user=user, family_id=family_id, expires_at=now + ACCESS_TOKEN_LIFETIME
            )
        return token

    def get(self, token: str) -> Session | None:
        key = _digest(token)
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


def get_database(request: Request) -> Database:
    return request.app.state.database


def get_session_store(request: Request) -> SessionStore:
    return request.app.state.sessions


def authenticated_user(
    credentials: Annotated[HTTPAuthorizationCredentials | None, Depends(bearer_auth)],
    sessions: Annotated[SessionStore, Depends(get_session_store)],
) -> User:
    if credentials is not None and credentials.scheme.lower() == "bearer":
        session = sessions.get(credentials.credentials)
        if session is not None:
            return session.user
    raise _unauthorized()


DatabaseDependency = Annotated[Database, Depends(get_database)]
SessionDependency = Annotated[SessionStore, Depends(get_session_store)]
UserDependency = Annotated[User, Depends(authenticated_user)]


@router.post("/login", response_model=TokenResponse)
def login(
    body: LoginRequest,
    response: Response,
    database: DatabaseDependency,
    sessions: SessionDependency,
) -> TokenResponse:
    response.headers["Cache-Control"] = "no-store"
    if not body.username or not body.password:
        raise HTTPException(status_code=400, detail="Username and password are required")
    try:
        user = database.get_user(body.username)
        if user is None:
            verify_password(DUMMY_PASSWORD_HASH, body.password)
            raise _unauthorized()
        if not verify_password(user.hashed_password, body.password):
            raise _unauthorized()
    except HTTPException:
        raise
    except (sqlite3.Error, ValueError) as error:
        logger.exception("authentication failed", exc_info=error)
        raise HTTPException(status_code=500, detail="Internal Server Error") from error

    refresh_token = _new_token()
    family_id = secrets.token_bytes(32)
    now = _now()
    try:
        database.refresh_tokens.create(
            user.id, _digest(refresh_token), family_id, now + REFRESH_IDLE_LIFETIME
        )
        access_token = sessions.create(user.model_copy(update={"hashed_password": ""}), family_id)
    except sqlite3.Error as error:
        logger.exception("create authentication session", exc_info=error)
        raise HTTPException(status_code=500, detail="Internal Server Error") from error
    return TokenResponse(access_token=access_token, refresh_token=refresh_token)


@router.post("/refresh", response_model=TokenResponse)
def refresh(
    body: RefreshRequest,
    response: Response,
    database: DatabaseDependency,
    sessions: SessionDependency,
) -> TokenResponse:
    response.headers["Cache-Control"] = "no-store"
    if not body.refresh_token:
        raise HTTPException(status_code=400, detail="Refresh token is required")
    new_refresh_token = _new_token()
    now = _now()
    try:
        rotation = database.refresh_tokens.rotate(
            _digest(body.refresh_token),
            _digest(new_refresh_token),
            now,
            now + REFRESH_IDLE_LIFETIME,
        )
    except (RefreshTokenReuseError, InvalidRefreshTokenError) as error:
        rotation = error.args[0] if error.args else None
        if rotation is not None:
            sessions.revoke_family(rotation.family_id)
        raise _unauthorized() from error
    except sqlite3.Error as error:
        logger.exception("rotate refresh token", exc_info=error)
        raise HTTPException(status_code=500, detail="Internal Server Error") from error

    access_token = sessions.create(rotation.user, rotation.family_id)
    return TokenResponse(access_token=access_token, refresh_token=new_refresh_token)


@router.post("/logout", status_code=status.HTTP_204_NO_CONTENT)
def logout(
    body: RefreshRequest, database: DatabaseDependency, sessions: SessionDependency
) -> Response:
    if not body.refresh_token:
        raise HTTPException(status_code=400, detail="Refresh token is required")
    try:
        family_id = database.refresh_tokens.revoke(_digest(body.refresh_token), _now())
    except InvalidRefreshTokenError:
        return Response(status_code=204, headers={"Cache-Control": "no-store"})
    except sqlite3.Error as error:
        logger.exception("revoke refresh token", exc_info=error)
        raise HTTPException(status_code=500, detail="Internal Server Error") from error
    sessions.revoke_family(family_id)
    return Response(status_code=204, headers={"Cache-Control": "no-store"})


@router.post("/logout-all", status_code=status.HTTP_204_NO_CONTENT)
def logout_all(
    user: UserDependency, database: DatabaseDependency, sessions: SessionDependency
) -> Response:
    try:
        database.refresh_tokens.revoke_user(user.id, _now())
    except sqlite3.Error as error:
        logger.exception("revoke all user sessions", exc_info=error)
        raise HTTPException(status_code=500, detail="Internal Server Error") from error
    sessions.revoke_user(user.id)
    return Response(status_code=204)


def _unauthorized() -> HTTPException:
    return HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Unauthorized",
        headers={"WWW-Authenticate": "Bearer"},
    )


def _new_token() -> str:
    return secrets.token_urlsafe(32)


def _digest(token: str) -> bytes:
    return hashlib.sha256(token.encode()).digest()


def _now() -> datetime:
    return datetime.now(UTC)
