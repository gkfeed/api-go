import pytest

from config import DEFAULT_ALLOWED_ORIGINS, load_settings, split_address


def test_load_settings_from_environment(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("GKFEED_ADDRESS", "127.0.0.1:9000")
    monkeypatch.setenv("GKFEED_DB_PATH", "/tmp/gkfeed.sqlite")
    monkeypatch.setenv("GKFEED_ALLOWED_ORIGINS", "https://one.example, https://two.example")

    settings = load_settings()

    assert settings.address == "127.0.0.1:9000"
    assert settings.database_path == "/tmp/gkfeed.sqlite"
    assert settings.allowed_origins == ("https://one.example", "https://two.example")


def test_load_settings_defaults(monkeypatch: pytest.MonkeyPatch) -> None:
    for name in ("GKFEED_ADDRESS", "GKFEED_DB_PATH", "GKFEED_ALLOWED_ORIGINS"):
        monkeypatch.delenv(name, raising=False)

    settings = load_settings()

    assert settings.address == ":8086"
    assert settings.database_path == "data/db.sqlite"
    assert settings.allowed_origins == DEFAULT_ALLOWED_ORIGINS
    assert split_address(settings.address) == ("0.0.0.0", 8086)
