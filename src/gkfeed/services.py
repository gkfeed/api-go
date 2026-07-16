from datetime import UTC
from email.utils import format_datetime
from xml.etree.ElementTree import Element, SubElement, tostring

from .models import FeedInput, Item

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


def generate_rss(items: list[Item]) -> bytes:
    root = Element("rss", version="2.0")
    channel = SubElement(root, "channel")
    SubElement(channel, "title").text = "GKFeed"
    SubElement(channel, "link").text = "/api/v1/feed"
    SubElement(channel, "description").text = "Personal GKFeed items"

    for item in items:
        element = SubElement(channel, "item")
        SubElement(element, "id").text = str(item.id)
        SubElement(element, "title").text = item.title
        SubElement(element, "link").text = item.link
        SubElement(element, "description").text = item.text
        date = item.date
        if date.tzinfo is None:
            date = date.replace(tzinfo=UTC)
        SubElement(element, "pubDate").text = format_datetime(date)

    return tostring(root, encoding="utf-8")
