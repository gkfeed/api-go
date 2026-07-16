import logging
import secrets
import sqlite3
from typing import Annotated

from fastapi import Depends, HTTPException, Request, status
from fastapi.security import HTTPBasic, HTTPBasicCredentials

from ..database import Database
from ..models import User

logger = logging.getLogger(__name__)
basic_auth = HTTPBasic(auto_error=False)


def get_database(request: Request) -> Database:
    return request.app.state.database


def authenticated_user(
    credentials: Annotated[HTTPBasicCredentials | None, Depends(basic_auth)],
    database: Annotated[Database, Depends(get_database)],
) -> User:
    unauthorized = HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Unauthorized",
        headers={"WWW-Authenticate": 'Basic realm="Restricted"'},
    )
    if credentials is None:
        raise unauthorized
    try:
        user = database.get_user(credentials.username)
    except sqlite3.Error as error:
        logger.exception("authentication failed", exc_info=error)
        raise HTTPException(status_code=500, detail="Internal Server Error") from error
    if user is None or not secrets.compare_digest(
        user.hashed_password.encode(), credentials.password.encode()
    ):
        raise unauthorized
    return user


DatabaseDependency = Annotated[Database, Depends(get_database)]
UserDependency = Annotated[User, Depends(authenticated_user)]
