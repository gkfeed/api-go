import sqlite3
from pathlib import Path

from fastapi.testclient import TestClient


def test_protected_route_requires_bearer_auth(client: TestClient) -> None:
    response = client.get("/api/v1/list")

    assert response.status_code == 401
    assert response.headers["www-authenticate"] == "Bearer"


def test_basic_credentials_are_rejected_by_protected_routes(client: TestClient) -> None:
    response = client.get("/api/v1/list", auth=("reader", "secret"))

    assert response.status_code == 401


def test_login_rejects_invalid_password(client: TestClient) -> None:
    response = client.post("/api/v1/auth/login", json={"username": "reader", "password": "wrong"})

    assert response.status_code == 401

    unknown = client.post("/api/v1/auth/login", json={"username": "missing", "password": "wrong"})
    assert unknown.status_code == 401


def test_login_refresh_and_logout(client: TestClient) -> None:
    login = client.post("/api/v1/auth/login", json={"username": "reader", "password": "secret"})
    assert login.status_code == 200
    assert login.headers["cache-control"] == "no-store"
    tokens = login.json()
    assert tokens["token_type"] == "Bearer"
    assert tokens["expires_in"] == 1800
    assert tokens["refresh_expires_in"] == 7_776_000
    assert (
        client.get(
            "/api/v1/list", headers={"Authorization": f"Bearer {tokens['access_token']}"}
        ).status_code
        == 200
    )

    refreshed = client.post("/api/v1/auth/refresh", json={"refresh_token": tokens["refresh_token"]})
    assert refreshed.status_code == 200
    rotated = refreshed.json()
    assert rotated["refresh_token"] != tokens["refresh_token"]

    reused = client.post("/api/v1/auth/refresh", json={"refresh_token": tokens["refresh_token"]})
    assert reused.status_code == 401
    assert (
        client.get(
            "/api/v1/list", headers={"Authorization": f"Bearer {rotated['access_token']}"}
        ).status_code
        == 401
    )


def test_logout_all_revokes_access_token(authenticated_client: TestClient) -> None:
    assert authenticated_client.post("/api/v1/auth/logout-all").status_code == 204
    assert authenticated_client.get("/api/v1/list").status_code == 401


def test_add_and_list_feed(authenticated_client: TestClient) -> None:
    response = authenticated_client.post(
        "/api/v1/add",
        json={"title": "Example", "type": "yt", "url": "https://youtube.com/example"},
    )

    assert response.status_code == 200
    assert response.json() == {
        "created": True,
        "item": {
            "id": 1,
            "title": "Example",
            "type": "yt",
            "url": "https://youtube.com/example",
            "userid": 1,
        },
    }
    assert authenticated_client.get("/api/v1/list").json() == [response.json()["item"]]


def test_add_rejects_unknown_fields(authenticated_client: TestClient) -> None:
    response = authenticated_client.post(
        "/api/v1/add", json={"title": "Example", "unexpected": True}
    )

    assert response.status_code == 400


def test_add_lazy_infers_and_normalizes_feed(authenticated_client: TestClient) -> None:
    response = authenticated_client.post(
        "/api/v1/add_lazy", json={"url": "https://tok.adminforge.de/@example"}
    )

    assert response.status_code == 200
    assert response.json()["item"] == {
        "id": 1,
        "title": "example",
        "type": "tiktok",
        "url": "https://www.tiktok.com/@example",
        "userid": 1,
    }


def test_add_lazy_rejects_invalid_url(authenticated_client: TestClient) -> None:
    response = authenticated_client.post(
        "/api/v1/add_lazy", json={"url": "https://example.com/feed"}
    )

    assert response.status_code == 400
    assert response.json()["detail"] == "invalid feed URL"


def test_delete_only_deletes_the_authenticated_users_feed(
    authenticated_client: TestClient, database_path: Path
) -> None:
    with sqlite3.connect(database_path) as connection:
        connection.execute(
            "INSERT INTO feed (id, title, url, type, user_id) VALUES (1, 'Other', '', '', 2)"
        )

    assert authenticated_client.delete("/api/v1/delete?id=1").status_code == 404
    assert authenticated_client.get("/api/v1/delete?id=1").status_code == 404


def test_items_are_paginated_and_deleted_items_are_hidden(
    authenticated_client: TestClient, database_path: Path
) -> None:
    with sqlite3.connect(database_path) as connection:
        connection.execute(
            "INSERT INTO feed (id, title, url, type, user_id) VALUES (1, 'Feed', '', '', 1)"
        )
        connection.executemany(
            "INSERT INTO item (id, feed_id, title, text, date, link) VALUES (?, 1, ?, '', ?, '')",
            (
                (1, "First", "2026-01-01T00:00:00+00:00"),
                (2, "Second", "2026-01-02T00:00:00+00:00"),
                (3, "Third", "2026-01-03T00:00:00+00:00"),
            ),
        )

    first_page = authenticated_client.get("/api/v1/get_items?limit=2")
    assert first_page.status_code == 200
    assert [item["id"] for item in first_page.json()["items"]] == [3, 2]
    assert first_page.json()["next_cursor"] == 2

    second_page = authenticated_client.get("/api/v1/get_items?limit=2&cursor=2")
    assert [item["id"] for item in second_page.json()["items"]] == [1]
    assert "next_cursor" not in second_page.json()

    deleted = authenticated_client.post("/api/v1/add_deleted_items", json={"itemIds": [3]})
    assert deleted.status_code == 200
    assert deleted.content == b""
    assert [
        item["id"] for item in authenticated_client.get("/api/v1/get_items").json()["items"]
    ] == [2, 1]


def test_public_item_and_rss_routes(
    authenticated_client: TestClient, client: TestClient, database_path: Path
) -> None:
    with sqlite3.connect(database_path) as connection:
        connection.execute(
            "INSERT INTO feed (id, title, url, type, user_id) VALUES (1, 'Feed', 'url', 'yt', 1)"
        )
        connection.execute(
            """
            INSERT INTO item (id, feed_id, title, text, date, link)
            VALUES (1, 1, 'Item', 'Description', '2026-01-01T00:00:00+00:00', 'link')
            """
        )

    item = client.get("/api/v1/item?id=1")
    assert item.status_code == 200
    assert item.json()["item"]["date"] == "2026-01-01T00:00:00Z"
    assert item.json()["feed"]["title"] == "Feed"

    rss = authenticated_client.get("/api/v1/feed")
    assert rss.status_code == 200
    assert rss.headers["content-type"].startswith("application/rss+xml")
    assert "<title>Item</title>" in rss.text


def test_query_validation(authenticated_client: TestClient, client: TestClient) -> None:
    assert client.get("/api/v1/item?id=0").status_code == 400
    assert authenticated_client.get("/api/v1/get_items?limit=0").status_code == 400
    assert authenticated_client.get("/api/v1/get_items?cursor=-1").status_code == 400
