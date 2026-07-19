import pytest

from services import create_feed_from_url
from services.opengraph import _parse_metadata


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


def test_parse_opengraph_metadata_with_fallbacks_and_relative_image() -> None:
    metadata = _parse_metadata(
        """
        <html>
          <head>
            <title>Fallback title</title>
            <meta property="og:title" content="Open Graph title">
            <meta name="description" content="Fallback description">
            <meta property="og:image" content="/preview.jpg">
            <meta property="og:site_name" content="Example">
            <meta property="og:type" content="article">
          </head>
        </html>
        """,
        "https://example.com/posts/1",
    )

    assert metadata.model_dump() == {
        "url": "https://example.com/posts/1",
        "title": "Open Graph title",
        "description": "Fallback description",
        "image": "https://example.com/preview.jpg",
        "site_name": "Example",
        "type": "article",
    }
