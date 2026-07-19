from .base import StrictModel


class OpenGraphMetadata(StrictModel):
    url: str
    title: str | None = None
    description: str | None = None
    image: str | None = None
    site_name: str | None = None
    type: str | None = None
