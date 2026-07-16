import uvicorn

from config import load_settings, split_address


def main() -> None:
    settings = load_settings()
    host, port = split_address(settings.address)
    uvicorn.run("api:app", host=host, port=port)


if __name__ == "__main__":
    main()
