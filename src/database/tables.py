from datetime import datetime

from sqlmodel import Field, SQLModel


class UserRecord(SQLModel, table=True):
    __tablename__ = "users"

    id: int | None = Field(default=None, primary_key=True)
    name: str
    password: str


class FeedRecord(SQLModel, table=True):
    __tablename__ = "feed"

    id: int | None = Field(default=None, primary_key=True)
    title: str
    url: str
    type: str
    user_id: int


class ItemRecord(SQLModel, table=True):
    __tablename__ = "item"

    id: int | None = Field(default=None, primary_key=True)
    feed_id: int
    title: str
    text: str
    date: datetime
    link: str


class DeletedItemRecord(SQLModel, table=True):
    __tablename__ = "deleted_items"

    user_id: int = Field(primary_key=True)
    item_id: int = Field(primary_key=True)


class RefreshTokenRecord(SQLModel, table=True):
    __tablename__ = "auth_refresh_tokens"

    id: int | None = Field(default=None, primary_key=True)
    token_hash: bytes = Field(unique=True, index=True)
    family_id: bytes = Field(index=True)
    user_id: int = Field(index=True)
    expires_at: int
    created_at: int
    revoked_at: int | None = None
    replaced_by: bytes | None = None
