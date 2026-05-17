from dataclasses import dataclass


@dataclass(frozen=True)
class Money:
    amount: int

    def __post_init__(self):
        if not isinstance(self.amount, int):
            raise TypeError("Price (Kano-Coin) must be an integer")
        # [INV-P01] Price (Giá của kịch bản) luôn phải lớn hơn 0
        if self.amount <= 0:
            raise ValueError("[INV-P01] Price must be greater than 0")

    def __str__(self) -> str:
        return f"{self.amount} Kano-Coins"
