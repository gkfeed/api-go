from sqlmodel import select

from models import User

from .session import SessionFactory
from .tables import UserRecord


class UserRepository:
    def __init__(self, sessions: SessionFactory) -> None:
        self.sessions = sessions

    def get(self, name: str) -> User | None:
        with self.sessions.create() as session:
            record = session.exec(select(UserRecord).where(UserRecord.name == name)).first()
        if record is None or record.id is None:
            return None
        return User(id=record.id, name=record.name, hashed_password=record.password)
