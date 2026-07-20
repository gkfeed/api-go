from models import User
from services.sessions import SessionStore


def test_session_store_creates_and_retrieves_sessions() -> None:
    sessions = SessionStore()
    user = User(id=1, name="reader", hashed_password="hash")

    token = sessions.create(user, b"family")

    session = sessions.get(token)
    assert session is not None
    assert session.user == user
    assert session.family_id == b"family"


def test_session_store_revokes_token_families() -> None:
    sessions = SessionStore()
    user = User(id=1, name="reader", hashed_password="hash")
    revoked = sessions.create(user, b"revoked")
    retained = sessions.create(user, b"retained")

    sessions.revoke_family(b"revoked")

    assert sessions.get(revoked) is None
    assert sessions.get(retained) is not None


def test_session_store_revokes_all_user_sessions() -> None:
    sessions = SessionStore()
    first_user = User(id=1, name="reader", hashed_password="hash")
    second_user = User(id=2, name="other", hashed_password="hash")
    revoked = sessions.create(first_user, b"first")
    retained = sessions.create(second_user, b"second")

    sessions.revoke_user(first_user.id)

    assert sessions.get(revoked) is None
    assert sessions.get(retained) is not None
