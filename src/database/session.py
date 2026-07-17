from collections.abc import Iterator
from contextlib import contextmanager

from sqlmodel import Session, create_engine


class SessionFactory:
    def __init__(self, path: str) -> None:
        self.path = path
        self.engine = create_engine(
            f"sqlite:///{path}",
            connect_args={"check_same_thread": False},
        )

    @contextmanager
    def create(self) -> Iterator[Session]:
        with Session(self.engine) as session:
            yield session
