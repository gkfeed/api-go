from models import FeedInput

FEED_TYPES_BY_PREFIX = (
    ("https://www.youtube.com/@", "yt"),
    ("https://www.instagram.com/", "inst"),
    ("https://tok.adminforge.de/@", "tiktok"),
    ("https://open.spotify.com/artist/", "spoti"),
    ("https://hdrezka.me/series/", "rezka"),
    ("https://hdrezka.me/films/", "rezka"),
    ("https://shikimori.one/animes/", "shiki"),
)


def create_feed_from_url(url: str) -> FeedInput:
    feed_type = next(
        (
            candidate_type
            for prefix, candidate_type in FEED_TYPES_BY_PREFIX
            if url.startswith(prefix) and len(url) > len(prefix)
        ),
        None,
    )
    if feed_type is None:
        raise ValueError("invalid feed URL")

    last_segment = url.rstrip("/").rsplit("/", 1)[-1]
    if feed_type in {"yt", "tiktok"}:
        title = url.split("@", 1)[1].split("/", 1)[0]
    elif feed_type == "rezka":
        title = last_segment.removesuffix(".html")
    else:
        title = last_segment

    normalized_url = url
    if feed_type == "tiktok":
        normalized_url = "https://www.tiktok.com/@" + url.split("@", 1)[1]

    return FeedInput(title=title, type=feed_type, url=normalized_url)
