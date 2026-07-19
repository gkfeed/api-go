import ipaddress
import socket
from html.parser import HTMLParser
from urllib.parse import urljoin, urlsplit

import httpx

from models import OpenGraphMetadata

MAX_BODY_BYTES = 1_000_000
MAX_REDIRECTS = 3


class _MetadataParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.metadata: dict[str, str] = {}
        self.title_parts: list[str] = []
        self.in_title = False

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        attributes = {key.lower(): value for key, value in attrs if value is not None}
        if tag.lower() == "title":
            self.in_title = True
        elif tag.lower() == "meta":
            name = (attributes.get("property") or attributes.get("name") or "").lower()
            content = attributes.get("content", "").strip()
            if name and content:
                self.metadata.setdefault(name, content)

    def handle_endtag(self, tag: str) -> None:
        if tag.lower() == "title":
            self.in_title = False

    def handle_data(self, data: str) -> None:
        if self.in_title:
            self.title_parts.append(data)


def _validate_public_url(url: str) -> None:
    parsed = urlsplit(url)
    if parsed.scheme not in {"http", "https"} or not parsed.hostname:
        raise ValueError("URL must use http or https")
    if parsed.username or parsed.password:
        raise ValueError("URL credentials are not allowed")

    try:
        port = parsed.port or (443 if parsed.scheme == "https" else 80)
        addresses = socket.getaddrinfo(parsed.hostname, port, type=socket.SOCK_STREAM)
    except (OSError, ValueError) as error:
        raise ValueError("URL host could not be resolved") from error

    if not addresses or any(not ipaddress.ip_address(item[4][0]).is_global for item in addresses):
        raise ValueError("URL must resolve to a public address")


def _parse_metadata(html: str, page_url: str) -> OpenGraphMetadata:
    parser = _MetadataParser()
    parser.feed(html)
    tags = parser.metadata

    title = tags.get("og:title") or tags.get("twitter:title")
    if not title:
        title = " ".join("".join(parser.title_parts).split()) or None
    image = tags.get("og:image") or tags.get("twitter:image")

    return OpenGraphMetadata(
        url=urljoin(page_url, tags.get("og:url", page_url)),
        title=title,
        description=(
            tags.get("og:description") or tags.get("twitter:description") or tags.get("description")
        ),
        image=urljoin(page_url, image) if image else None,
        site_name=tags.get("og:site_name"),
        type=tags.get("og:type"),
    )


def fetch_open_graph(url: str) -> OpenGraphMetadata:
    headers = {"User-Agent": "GKFeed/1.0 (+OpenGraph preview)"}
    timeout = httpx.Timeout(10, connect=5)

    with httpx.Client(headers=headers, timeout=timeout, follow_redirects=False) as client:
        for _ in range(MAX_REDIRECTS + 1):
            _validate_public_url(url)
            with client.stream("GET", url) as response:
                if response.is_redirect:
                    location = response.headers.get("location")
                    if not location:
                        raise ValueError("invalid redirect")
                    url = urljoin(url, location)
                    continue

                response.raise_for_status()
                content_type = response.headers.get("content-type", "").lower()
                if "text/html" not in content_type and "application/xhtml+xml" not in content_type:
                    raise ValueError("URL did not return HTML")

                body = bytearray()
                for chunk in response.iter_bytes():
                    body.extend(chunk)
                    if len(body) > MAX_BODY_BYTES:
                        raise ValueError("HTML response is too large")
                return _parse_metadata(body.decode(response.encoding or "utf-8", "replace"), url)

    raise ValueError("too many redirects")
