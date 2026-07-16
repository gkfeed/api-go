import sqlite3
from collections.abc import Iterator
from pathlib import Path

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from gkfeed.api import create_app
from gkfeed.config import Settings

SCHEMA = (
    "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, password TEXT)",
    "CREATE TABLE feed (id INTEGER PRIMARY KEY, title TEXT, url TEXT, type TEXT, user_id INTEGER)",
    """
    CREATE TABLE item (
        id INTEGER PRIMARY KEY,
        feed_id INTEGER,
        title TEXT,
        text TEXT,
        date DATETIME,
        link TEXT
    )
    """,
    "CREATE TABLE deleted_items (user_id INTEGER, item_id INTEGER)",
    "INSERT INTO users (id, name, password) VALUES (1, 'reader', 'secret')",
    "INSERT INTO users (id, name, password) VALUES (2, 'other', 'secret')",
)


@pytest.fixture
def database_path(tmp_path: Path) -> Path:
    path = tmp_path / "db.sqlite"
    with sqlite3.connect(path) as connection:
        for statement in SCHEMA:
            connection.execute(statement)
    return path


@pytest.fixture
def fastapi_app(database_path: Path) -> FastAPI:
    return create_app(Settings(":8086", str(database_path), ("http://localhost",)))


@pytest.fixture
def client(fastapi_app: FastAPI) -> Iterator[TestClient]:
    with TestClient(fastapi_app) as test_client:
        yield test_client


@pytest.fixture
def authenticated_client(client: TestClient) -> TestClient:
    client.auth = ("reader", "secret")
    return client
