from typing import Annotated

from fastapi import Depends, Request

from database import Database


def get_database(request: Request) -> Database:
    return request.app.state.database


DatabaseDependency = Annotated[Database, Depends(get_database)]
