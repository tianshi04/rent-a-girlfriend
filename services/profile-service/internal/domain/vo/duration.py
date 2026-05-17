from dataclasses import dataclass


@dataclass(frozen=True)
class Duration:
    minutes: int

    def __post_init__(self):
        if not isinstance(self.minutes, int):
            raise TypeError("Duration must be an integer")
        # [INV-P02] Duration phải nằm trong các mốc quy định của nền tảng (VD: 60, 120, 180 phút)
        if self.minutes not in (60, 120, 180):
            raise ValueError("[INV-P02] Duration must be 60, 120, or 180 minutes")

    def __str__(self) -> str:
        return f"{self.minutes} minutes"
