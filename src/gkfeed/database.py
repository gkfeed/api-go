import sqlite3
from collections.abc import Iterator, Sequence
from contextlib import contextmanager
from datetime import datetime

from .models import Feed, FeedInput, Item, User


class Database:
    def __init__(self, path: str) -> None:
        self.path = path

    @contextmanager
    def connect(self) -> Iterator[sqlite3.Connection]:
        connection = sqlite3.connect(self.path)
        connection.row_factory = sqlite3.Row
        try:
            yield connection
        finally:
            connection.close()

    def get_user(self, name: str) -> User | None:
        with self.connect() as connection:
            row = connection.execute(
                "SELECT id, name, password FROM users WHERE name = ?", (name,)
            ).fetchone()
        if row is None:
            return None
        return User(id=row["id"], name=row["name"], hashed_password=row["password"])

    def get_user_feeds(self, user_id: int) -> list[Feed]:
        with self.connect() as connection:
            rows = connection.execute(
                "SELECT id, title, url, type, user_id FROM feed WHERE user_id = ?", (user_id,)
            ).fetchall()
        return [self._feed(row) for row in rows]

    def add_feed(self, feed: FeedInput, user_id: int) -> Feed:
        with self.connect() as connection:
            cursor = connection.execute(
                "INSERT INTO feed (title, type, url, user_id) VALUES (?, ?, ?, ?)",
                (feed.title, feed.type, feed.url, user_id),
            )
            connection.commit()
            feed_id = cursor.lastrowid
        if feed_id is None:
            raise RuntimeError("SQLite did not return the inserted feed ID")
        return Feed(id=feed_id, title=feed.title, type=feed.type, url=feed.url, userid=user_id)

    def get_feed(self, feed_id: int) -> Feed | None:
        with self.connect() as connection:
            row = connection.execute(
                "SELECT id, title, url, type, user_id FROM feed WHERE id = ?", (feed_id,)
            ).fetchone()
        return self._feed(row) if row is not None else None

    def delete_feed(self, feed_id: int) -> None:
        with self.connect() as connection:
            connection.execute("DELETE FROM feed WHERE id = ?", (feed_id,))
            connection.commit()

    def get_user_items(
        self, user_id: int, *, cursor: int | None = None, limit: int | None = None
    ) -> list[Item]:
        query = """
            SELECT item.id, item.feed_id, item.title, item.text, item.date, item.link
            FROM item
            JOIN feed ON item.feed_id = feed.id
            WHERE feed.user_id = ?
              AND item.id NOT IN (
                SELECT item_id FROM deleted_items WHERE user_id = ?
              )
        """
        parameters: list[int] = [user_id, user_id]
        if cursor is not None:
            query += " AND item.id < ?"
            parameters.append(cursor)
        if limit is not None:
            query += " ORDER BY item.id DESC LIMIT ?"
            parameters.append(limit)

        with self.connect() as connection:
            rows = connection.execute(query, parameters).fetchall()
        return [self._item(row) for row in rows]

    def add_deleted_items(self, user_id: int, item_ids: Sequence[int]) -> None:
        with self.connect() as connection:
            connection.executemany(
                "INSERT INTO deleted_items (user_id, item_id) VALUES (?, ?)",
                ((user_id, item_id) for item_id in item_ids),
            )
            connection.commit()

    def get_item(self, item_id: int) -> Item | None:
        with self.connect() as connection:
            row = connection.execute(
                """
                SELECT item.id, item.feed_id, item.title, item.text, item.date, item.link
                FROM item WHERE item.id = ?
                """,
                (item_id,),
            ).fetchone()
        return self._item(row) if row is not None else None

    @staticmethod
    def _feed(row: sqlite3.Row) -> Feed:
        return Feed(
            id=row["id"],
            title=row["title"],
            type=row["type"],
            url=row["url"],
            userid=row["user_id"],
        )

    @staticmethod
    def _item(row: sqlite3.Row) -> Item:
        date = row["date"]
        if isinstance(date, str):
            date = datetime.fromisoformat(date.replace("Z", "+00:00"))
        return Item(
            id=row["id"],
            feed_id=row["feed_id"],
            title=row["title"],
            text=row["text"],
            date=date,
            link=row["link"],
        )
