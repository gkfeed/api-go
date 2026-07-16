from collections.abc import Sequence

from sqlmodel import select

from ..models import Item
from .session import SessionFactory
from .tables import DeletedItemRecord, FeedRecord, ItemRecord


class ItemRepository:
    def __init__(self, sessions: SessionFactory) -> None:
        self.sessions = sessions

    def list_for_user(
        self, user_id: int, *, cursor: int | None = None, limit: int | None = None
    ) -> list[Item]:
        with self.sessions.create() as session:
            hidden_ids = select(DeletedItemRecord.item_id).where(
                DeletedItemRecord.user_id == user_id
            )
            statement = (
                select(ItemRecord)
                .join(FeedRecord, ItemRecord.feed_id == FeedRecord.id)
                .where(
                    FeedRecord.user_id == user_id,
                    ItemRecord.id.not_in(hidden_ids),
                )
            )
            if cursor is not None:
                statement = statement.where(ItemRecord.id < cursor)
            if limit is not None:
                statement = statement.order_by(ItemRecord.id.desc()).limit(limit)
            records = session.exec(statement).all()
        return [_item_from_record(record) for record in records]

    def hide_for_user(self, user_id: int, item_ids: Sequence[int]) -> None:
        with self.sessions.create() as session:
            session.add_all(
                DeletedItemRecord(user_id=user_id, item_id=item_id) for item_id in item_ids
            )
            session.commit()

    def get(self, item_id: int) -> Item | None:
        with self.sessions.create() as session:
            record = session.get(ItemRecord, item_id)
            return _item_from_record(record) if record is not None else None


def _item_from_record(record: ItemRecord) -> Item:
    if record.id is None:
        raise ValueError("item record has no ID")
    return Item(
        id=record.id,
        feed_id=record.feed_id,
        title=record.title,
        text=record.text,
        date=record.date,
        link=record.link,
    )
