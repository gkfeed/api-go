import pytest

from gkfeed.services import create_feed_from_url


@pytest.mark.parametrize(
    ("url", "title", "feed_type", "normalized_url"),
    (
        (
            "https://www.youtube.com/@example/videos",
            "example",
            "yt",
            "https://www.youtube.com/@example/videos",
        ),
        (
            "https://tok.adminforge.de/@example",
            "example",
            "tiktok",
            "https://www.tiktok.com/@example",
        ),
        (
            "https://hdrezka.me/films/drama/example.html",
            "example",
            "rezka",
            "https://hdrezka.me/films/drama/example.html",
        ),
    ),
)
def test_create_feed_from_url(url: str, title: str, feed_type: str, normalized_url: str) -> None:
    feed = create_feed_from_url(url)

    assert (feed.title, feed.type, feed.url) == (title, feed_type, normalized_url)


def test_create_feed_from_url_rejects_incomplete_url() -> None:
    with pytest.raises(ValueError, match="invalid feed URL"):
        create_feed_from_url("https://www.youtube.com/@")
