from typing import Annotated

from fastapi import APIRouter, HTTPException, Query, Response

from ...models import DeletedItemsInput, ItemsPage, ItemWithFeed
from ..dependencies import DatabaseDependency, UserDependency

router = APIRouter()


@router.post("/add_deleted_items", status_code=200)
def add_deleted_items(
    request: DeletedItemsInput, user: UserDependency, database: DatabaseDependency
) -> Response:
    database.add_deleted_items(user.id, request.item_ids)
    return Response(status_code=200)


@router.get("/get_items", response_model=ItemsPage, response_model_exclude_none=True)
def get_items(
    user: UserDependency,
    database: DatabaseDependency,
    limit: Annotated[int, Query(gt=0)] = 100,
    cursor: Annotated[int | None, Query(ge=0)] = None,
) -> ItemsPage:
    items = database.get_user_items(user.id, cursor=cursor, limit=limit + 1)
    next_cursor = items[limit - 1].id if len(items) > limit else None
    return ItemsPage(items=items[:limit], next_cursor=next_cursor)


@router.get("/item", response_model=ItemWithFeed)
def get_item(database: DatabaseDependency, id: Annotated[int, Query(gt=0)]) -> ItemWithFeed:
    item = database.get_item(id)
    if item is None:
        raise HTTPException(status_code=404, detail="Not Found")
    feed = database.get_feed(item.feed_id)
    if feed is None:
        raise HTTPException(status_code=404, detail="Not Found")
    return ItemWithFeed(item=item, feed=feed)
