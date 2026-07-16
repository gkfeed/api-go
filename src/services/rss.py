from datetime import UTC
from email.utils import format_datetime
from xml.etree.ElementTree import Element, SubElement, tostring

from models import Item


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
