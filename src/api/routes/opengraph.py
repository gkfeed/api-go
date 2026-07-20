from typing import Annotated

import httpx
from fastapi import APIRouter, HTTPException, Query

from models import OpenGraphMetadata
from services import fetch_open_graph

from ..auth.dependencies import UserDependency

router = APIRouter()


@router.get("/opengraph", response_model=OpenGraphMetadata, response_model_exclude_none=True)
def opengraph_metadata(
    user: UserDependency,
    url: Annotated[str, Query(min_length=1, max_length=2048)],
) -> OpenGraphMetadata:
    del user
    try:
        return fetch_open_graph(url)
    except ValueError as error:
        raise HTTPException(status_code=400, detail=str(error)) from error
    except httpx.HTTPError as error:
        raise HTTPException(status_code=502, detail="could not fetch URL") from error
