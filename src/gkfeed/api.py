import logging
import secrets
import sqlite3
from typing import Annotated

from fastapi import Depends, FastAPI, HTTPException, Query, Response, status
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import PlainTextResponse
from fastapi.security import HTTPBasic, HTTPBasicCredentials

from .config import Settings, load_settings
from .database import Database
from .models import (
    DeletedItemsInput,
    Feed,
    FeedDeletion,
    FeedInput,
    FeedMutation,
    ItemsPage,
    ItemWithFeed,
    LazyFeedInput,
    User,
)
from .services import create_feed_from_url, generate_rss

logger = logging.getLogger(__name__)
basic_auth = HTTPBasic(auto_error=False)


def create_app(settings: Settings | None = None) -> FastAPI:
    configuration = settings or load_settings()
    app = FastAPI(title="GKFeed API", version="1.0.0")
    app.state.database = Database(configuration.database_path)
    app.add_middleware(
        CORSMiddleware,
        allow_origins=list(configuration.allowed_origins),
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    @app.exception_handler(RequestValidationError)
    async def validation_error_handler(
        _request: object, error: RequestValidationError
    ) -> PlainTextResponse:
        first_error = error.errors()[0]
        location = first_error.get("loc", ())
        if location and location[0] == "query":
            parameter = str(location[-1])
            message = f"Invalid {parameter}"
        else:
            message = "Invalid request body"
        return PlainTextResponse(message, status_code=400)

    def get_database() -> Database:
        return app.state.database

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

    @app.get("/api/v1/list", response_model=list[Feed])
    def list_feeds(user: UserDependency, database: DatabaseDependency) -> list[Feed]:
        return database.get_user_feeds(user.id)

    @app.get("/api/v1/feed", response_class=Response)
    def rss_feed(user: UserDependency, database: DatabaseDependency) -> Response:
        return Response(
            content=generate_rss(database.get_user_items(user.id)),
            media_type="application/rss+xml",
        )

    @app.post("/api/v1/add", response_model=FeedMutation)
    def add_feed(
        feed_input: FeedInput, user: UserDependency, database: DatabaseDependency
    ) -> FeedMutation:
        return FeedMutation(created=True, item=database.add_feed(feed_input, user.id))

    @app.post("/api/v1/add_lazy", response_model=FeedMutation)
    def add_feed_lazy(
        feed_input: LazyFeedInput, user: UserDependency, database: DatabaseDependency
    ) -> FeedMutation:
        try:
            inferred_feed = create_feed_from_url(feed_input.url)
        except ValueError as error:
            raise HTTPException(status_code=400, detail=str(error)) from error
        return FeedMutation(created=True, item=database.add_feed(inferred_feed, user.id))

    @app.delete("/api/v1/delete", response_model=FeedDeletion)
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

    @app.get("/api/v1/delete", include_in_schema=False)
    def reject_get_delete() -> None:
        raise HTTPException(status_code=404, detail="Not Found")

    @app.post("/api/v1/add_deleted_items", status_code=200)
    def add_deleted_items(
        request: DeletedItemsInput, user: UserDependency, database: DatabaseDependency
    ) -> Response:
        database.add_deleted_items(user.id, request.item_ids)
        return Response(status_code=200)

    @app.get("/api/v1/get_items", response_model=ItemsPage, response_model_exclude_none=True)
    def get_items(
        user: UserDependency,
        database: DatabaseDependency,
        limit: Annotated[int, Query(gt=0)] = 100,
        cursor: Annotated[int | None, Query(ge=0)] = None,
    ) -> ItemsPage:
        items = database.get_user_items(user.id, cursor=cursor, limit=limit + 1)
        next_cursor = items[limit - 1].id if len(items) > limit else None
        return ItemsPage(items=items[:limit], next_cursor=next_cursor)

    @app.get("/api/v1/item", response_model=ItemWithFeed)
    def get_item(database: DatabaseDependency, id: Annotated[int, Query(gt=0)]) -> ItemWithFeed:
        item = database.get_item(id)
        if item is None:
            raise HTTPException(status_code=404, detail="Not Found")
        feed = database.get_feed(item.feed_id)
        if feed is None:
            raise HTTPException(status_code=404, detail="Not Found")
        return ItemWithFeed(item=item, feed=feed)

    return app


app = create_app()
