import sqlite3
from dataclasses import dataclass
from datetime import datetime

from models import User

from .session import SessionFactory


class InvalidRefreshTokenError(Exception):
    pass


class RefreshTokenReuseError(Exception):
    pass


@dataclass(frozen=True)
class RefreshRotation:
    user: User
    family_id: bytes


class RefreshTokenRepository:
    def __init__(self, sessions: SessionFactory) -> None:
        self.path = sessions.path
        self._initialize()

    def _connect(self) -> sqlite3.Connection:
        connection = sqlite3.connect(self.path)
        connection.row_factory = sqlite3.Row
        return connection

    def _initialize(self) -> None:
        with self._connect() as connection:
            connection.executescript(
                """
                CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    token_hash BLOB NOT NULL UNIQUE,
                    family_id BLOB NOT NULL,
                    user_id INTEGER NOT NULL,
                    expires_at INTEGER NOT NULL,
                    created_at INTEGER NOT NULL,
                    revoked_at INTEGER,
                    replaced_by BLOB
                );
                CREATE INDEX IF NOT EXISTS auth_refresh_tokens_family
                    ON auth_refresh_tokens (family_id);
                CREATE INDEX IF NOT EXISTS auth_refresh_tokens_user
                    ON auth_refresh_tokens (user_id);
                """
            )

    def create(
        self, user_id: int, token_hash: bytes, family_id: bytes, expires_at: datetime
    ) -> None:
        with self._connect() as connection:
            connection.execute(
                """INSERT INTO auth_refresh_tokens
                       (token_hash, family_id, user_id, expires_at, created_at)
                   VALUES (?, ?, ?, ?, ?)""",
                (token_hash, family_id, user_id, int(expires_at.timestamp()), _timestamp()),
            )

    def rotate(
        self,
        old_hash: bytes,
        new_hash: bytes,
        current_time: datetime,
        expires_at: datetime,
    ) -> RefreshRotation:
        now = int(current_time.timestamp())
        with self._connect() as connection:
            connection.execute("BEGIN IMMEDIATE")
            row = connection.execute(
                """SELECT token.family_id, token.user_id, users.name, token.expires_at,
                          token.revoked_at, token.replaced_by
                     FROM auth_refresh_tokens AS token
                     JOIN users ON users.id = token.user_id
                    WHERE token.token_hash = ?""",
                (old_hash,),
            ).fetchone()
            if row is None:
                raise InvalidRefreshTokenError

            rotation = RefreshRotation(
                User(id=row["user_id"], name=row["name"], hashed_password=""),
                bytes(row["family_id"]),
            )
            if row["revoked_at"] is not None or row["replaced_by"] is not None:
                self._revoke_family(connection, rotation.family_id, now)
                connection.commit()
                raise RefreshTokenReuseError(rotation)
            if row["expires_at"] <= now:
                self._revoke_family(connection, rotation.family_id, now)
                connection.commit()
                raise InvalidRefreshTokenError(rotation)

            updated = connection.execute(
                """UPDATE auth_refresh_tokens
                      SET revoked_at = ?, replaced_by = ?
                    WHERE token_hash = ? AND revoked_at IS NULL AND replaced_by IS NULL""",
                (now, new_hash, old_hash),
            )
            if updated.rowcount != 1:
                self._revoke_family(connection, rotation.family_id, now)
                connection.commit()
                raise RefreshTokenReuseError(rotation)
            connection.execute(
                """INSERT INTO auth_refresh_tokens
                       (token_hash, family_id, user_id, expires_at, created_at)
                   VALUES (?, ?, ?, ?, ?)""",
                (
                    new_hash,
                    rotation.family_id,
                    rotation.user.id,
                    int(expires_at.timestamp()),
                    now,
                ),
            )
            connection.commit()
            return rotation

    def revoke(self, token_hash: bytes, current_time: datetime) -> bytes:
        with self._connect() as connection:
            row = connection.execute(
                "SELECT family_id FROM auth_refresh_tokens WHERE token_hash = ?", (token_hash,)
            ).fetchone()
            if row is None:
                raise InvalidRefreshTokenError
            family_id = bytes(row["family_id"])
            self._revoke_family(connection, family_id, int(current_time.timestamp()))
            return family_id

    def revoke_user(self, user_id: int, current_time: datetime) -> None:
        with self._connect() as connection:
            connection.execute(
                """UPDATE auth_refresh_tokens
                      SET revoked_at = COALESCE(revoked_at, ?)
                    WHERE user_id = ?""",
                (int(current_time.timestamp()), user_id),
            )

    @staticmethod
    def _revoke_family(connection: sqlite3.Connection, family_id: bytes, now: int) -> None:
        connection.execute(
            """UPDATE auth_refresh_tokens
                  SET revoked_at = COALESCE(revoked_at, ?)
                WHERE family_id = ?""",
            (now, family_id),
        )


def _timestamp() -> int:
    return int(datetime.now().timestamp())
