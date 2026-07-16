from pydantic import BaseModel

from .base import StrictModel


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


class LazyFeedInput(StrictModel):
    url: str = ""


class FeedMutation(BaseModel):
    created: bool
    item: Feed


class FeedDeletion(BaseModel):
    deleted: bool
    item: Feed
