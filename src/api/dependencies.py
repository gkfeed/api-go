from typing import Annotated

from fastapi import Depends, Request

from database import Database
from models import User

from .auth import authenticated_user


def get_database(request: Request) -> Database:
    return request.app.state.database


DatabaseDependency = Annotated[Database, Depends(get_database)]
UserDependency = Annotated[User, Depends(authenticated_user)]
