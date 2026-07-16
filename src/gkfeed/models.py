from datetime import datetime

from pydantic import BaseModel, ConfigDict, Field, field_serializer


class StrictModel(BaseModel):
    model_config = ConfigDict(extra="forbid")


class User(BaseModel):
    id: int
    name: str
    hashed_password: str = Field(exclude=True)


class FeedInput(StrictModel):
    id: int = 0
    title: str = ""
    type: str = ""
    url: str = ""
    userid: int = 0


class Feed(BaseModel):
    id: int
    title: str
    type: str
    url: str
    userid: int


class Item(BaseModel):
    id: int
    feed_id: int
    title: str
    text: str
    date: datetime
    link: str

    @field_serializer("date", when_used="json")
    def serialize_date(self, value: datetime) -> str:
        return value.isoformat().replace("+00:00", "Z")


class LazyFeedInput(StrictModel):
    url: str = ""


class DeletedItemsInput(StrictModel):
    item_ids: list[int] = Field(default_factory=list, alias="itemIds")


class FeedMutation(BaseModel):
    created: bool
    item: Feed


class FeedDeletion(BaseModel):
    deleted: bool
    item: Feed


class ItemsPage(BaseModel):
    items: list[Item]
    next_cursor: int | None = None

    model_config = ConfigDict(exclude_none=True)


class ItemWithFeed(BaseModel):
    item: Item
    feed: Feed
