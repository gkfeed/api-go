import logging
import secrets
import sqlite3
from datetime import UTC, datetime

from fastapi import APIRouter, HTTPException, Response, status

from database.refresh_tokens import InvalidRefreshTokenError, RefreshTokenReuseError
from services.passwords import DUMMY_PASSWORD_HASH, verify_password
from services.sessions import REFRESH_IDLE_LIFETIME, digest_token, generate_token

from ..dependencies import DatabaseDependency
from .dependencies import SessionDependency, UserDependency, unauthorized
from .schemas import LoginRequest, RefreshRequest, TokenResponse

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/auth", tags=["auth"])


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
            raise unauthorized()
        if not verify_password(user.hashed_password, body.password):
            raise unauthorized()
    except HTTPException:
        raise
    except (sqlite3.Error, ValueError) as error:
        logger.exception("authentication failed", exc_info=error)
        raise HTTPException(status_code=500, detail="Internal Server Error") from error

    refresh_token = generate_token()
    family_id = secrets.token_bytes(32)
    now = _now()
    try:
        database.refresh_tokens.create(
            user.id,
            digest_token(refresh_token),
            family_id,
            now + REFRESH_IDLE_LIFETIME,
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
    new_refresh_token = generate_token()
    now = _now()
    try:
        rotation = database.refresh_tokens.rotate(
            digest_token(body.refresh_token),
            digest_token(new_refresh_token),
            now,
            now + REFRESH_IDLE_LIFETIME,
        )
    except (RefreshTokenReuseError, InvalidRefreshTokenError) as error:
        rotation = error.args[0] if error.args else None
        if rotation is not None:
            sessions.revoke_family(rotation.family_id)
        raise unauthorized() from error
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
        family_id = database.refresh_tokens.revoke(digest_token(body.refresh_token), _now())
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


def _now() -> datetime:
    return datetime.now(UTC)
