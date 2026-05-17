from dataclasses import dataclass


@dataclass(frozen=True)
class MediaUrl:
    url: str

    def __post_init__(self):
        if not isinstance(self.url, str):
            raise TypeError("Media URL must be a string")

        clean_url = self.url.strip()
        if not clean_url:
            raise ValueError("URL cannot be empty")

        if not (clean_url.startswith("http://") or clean_url.startswith("https://")):
            raise ValueError("URL must start with http:// or https://")

        object.__setattr__(self, "url", clean_url)

    def __str__(self) -> str:
        return self.url
