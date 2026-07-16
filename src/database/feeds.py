from sqlmodel import select

from models import Feed, FeedInput

from .session import SessionFactory
from .tables import FeedRecord


class FeedRepository:
    def __init__(self, sessions: SessionFactory) -> None:
        self.sessions = sessions

    def list_for_user(self, user_id: int) -> list[Feed]:
        with self.sessions.create() as session:
            records = session.exec(select(FeedRecord).where(FeedRecord.user_id == user_id)).all()
        return [_feed_from_record(record) for record in records]

    def add(self, feed: FeedInput, user_id: int) -> Feed:
        record = FeedRecord(
            title=feed.title,
            type=feed.type,
            url=feed.url,
            user_id=user_id,
        )
        with self.sessions.create() as session:
            session.add(record)
            session.commit()
            session.refresh(record)
            if record.id is None:
                raise RuntimeError("SQLite did not return the inserted feed ID")
            return Feed(
                id=record.id,
                title=record.title,
                type=record.type,
                url=record.url,
                userid=record.user_id,
            )

    def get(self, feed_id: int) -> Feed | None:
        with self.sessions.create() as session:
            record = session.get(FeedRecord, feed_id)
            return _feed_from_record(record) if record is not None else None

    def delete(self, feed_id: int) -> None:
        with self.sessions.create() as session:
            record = session.get(FeedRecord, feed_id)
            if record is not None:
                session.delete(record)
                session.commit()


def _feed_from_record(record: FeedRecord) -> Feed:
    if record.id is None:
        raise ValueError("feed record has no ID")
    return Feed(
        id=record.id,
        title=record.title,
        type=record.type,
        url=record.url,
        userid=record.user_id,
    )
