from collections.abc import Sequence

from ..models import Feed, FeedInput, Item, User
from .feeds import FeedRepository
from .items import ItemRepository
from .session import SessionFactory
from .users import UserRepository


class Database:
    """Compatibility facade over domain-specific repositories."""

    def __init__(self, path: str) -> None:
        sessions = SessionFactory(path)
        self.users = UserRepository(sessions)
        self.feeds = FeedRepository(sessions)
        self.items = ItemRepository(sessions)

    def get_user(self, name: str) -> User | None:
        return self.users.get(name)

    def get_user_feeds(self, user_id: int) -> list[Feed]:
        return self.feeds.list_for_user(user_id)

    def add_feed(self, feed: FeedInput, user_id: int) -> Feed:
        return self.feeds.add(feed, user_id)

    def get_feed(self, feed_id: int) -> Feed | None:
        return self.feeds.get(feed_id)

    def delete_feed(self, feed_id: int) -> None:
        self.feeds.delete(feed_id)

    def get_user_items(
        self, user_id: int, *, cursor: int | None = None, limit: int | None = None
    ) -> list[Item]:
        return self.items.list_for_user(user_id, cursor=cursor, limit=limit)

    def add_deleted_items(self, user_id: int, item_ids: Sequence[int]) -> None:
        self.items.hide_for_user(user_id, item_ids)

    def get_item(self, item_id: int) -> Item | None:
        return self.items.get(item_id)
