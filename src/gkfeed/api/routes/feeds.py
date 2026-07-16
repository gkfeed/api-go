from typing import Annotated

from fastapi import APIRouter, HTTPException, Query, Response

from ...models import Feed, FeedDeletion, FeedInput, FeedMutation, LazyFeedInput
from ...services import create_feed_from_url, generate_rss
from ..dependencies import DatabaseDependency, UserDependency

router = APIRouter()


@router.get("/list", response_model=list[Feed])
def list_feeds(user: UserDependency, database: DatabaseDependency) -> list[Feed]:
    return database.get_user_feeds(user.id)


@router.get("/feed", response_class=Response)
def rss_feed(user: UserDependency, database: DatabaseDependency) -> Response:
    return Response(
        content=generate_rss(database.get_user_items(user.id)),
        media_type="application/rss+xml",
    )


@router.post("/add", response_model=FeedMutation)
def add_feed(
    feed_input: FeedInput, user: UserDependency, database: DatabaseDependency
) -> FeedMutation:
    return FeedMutation(created=True, item=database.add_feed(feed_input, user.id))


@router.post("/add_lazy", response_model=FeedMutation)
def add_feed_lazy(
    feed_input: LazyFeedInput, user: UserDependency, database: DatabaseDependency
) -> FeedMutation:
    try:
        inferred_feed = create_feed_from_url(feed_input.url)
    except ValueError as error:
        raise HTTPException(status_code=400, detail=str(error)) from error
    return FeedMutation(created=True, item=database.add_feed(inferred_feed, user.id))


@router.delete("/delete", response_model=FeedDeletion)
def delete_feed(
    user: UserDependency,
    database: DatabaseDependency,
    id: Annotated[int, Query(gt=0)],
) -> FeedDeletion:
    feed = database.get_feed(id)
    if feed is None or feed.userid != user.id:
        raise HTTPException(status_code=404, detail="Not Found")
    database.delete_feed(id)
    return FeedDeletion(deleted=True, item=feed)


@router.get("/delete", include_in_schema=False)
def reject_get_delete() -> None:
    raise HTTPException(status_code=404, detail="Not Found")
