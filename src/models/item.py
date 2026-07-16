from datetime import datetime

from pydantic import BaseModel, ConfigDict, Field, field_serializer

from .base import StrictModel
from .feed import Feed


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


class DeletedItemsInput(StrictModel):
    item_ids: list[int] = Field(default_factory=list, alias="itemIds")


class ItemsPage(BaseModel):
    items: list[Item]
    next_cursor: int | None = None

    model_config = ConfigDict(exclude_none=True)


class ItemWithFeed(BaseModel):
    item: Item
    feed: Feed
