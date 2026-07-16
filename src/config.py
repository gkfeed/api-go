import os
from dataclasses import dataclass

DEFAULT_ALLOWED_ORIGINS = (
    "http://localhost",
    "http://localhost:4200",
    "http://localhost:8086",
)


@dataclass(frozen=True, slots=True)
class Settings:
    address: str
    database_path: str
    allowed_origins: tuple[str, ...]


def _value_or_default(name: str, fallback: str) -> str:
    return os.environ.get(name, "").strip() or fallback


def load_settings() -> Settings:
    origins_value = os.environ.get("GKFEED_ALLOWED_ORIGINS", "")
    origins = (
        tuple(origin.strip() for origin in origins_value.split(",") if origin.strip())
        if origins_value
        else DEFAULT_ALLOWED_ORIGINS
    )
    return Settings(
        address=_value_or_default("GKFEED_ADDRESS", ":8086"),
        database_path=_value_or_default("GKFEED_DB_PATH", "data/db.sqlite"),
        allowed_origins=origins,
    )


def split_address(address: str) -> tuple[str, int]:
    host, separator, raw_port = address.rpartition(":")
    if not separator or not raw_port:
        raise ValueError(f"invalid GKFEED_ADDRESS: {address!r}")
    return host or "0.0.0.0", int(raw_port)
