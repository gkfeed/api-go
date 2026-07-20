from typing import Annotated

from fastapi import Depends, HTTPException, Request, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from models import User
from services.sessions import SessionStore

bearer_auth = HTTPBearer(auto_error=False)


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
    raise unauthorized()


def unauthorized() -> HTTPException:
    return HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Unauthorized",
        headers={"WWW-Authenticate": "Bearer"},
    )


SessionDependency = Annotated[SessionStore, Depends(get_session_store)]
UserDependency = Annotated[User, Depends(authenticated_user)]
