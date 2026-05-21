from internal.domain.vo import Money


class CurrencyExchangeService:
    RATE_COIN_TO_VND = 1000  # 1 Kano-Coin = 1,000 VND

    @staticmethod
    def coin_to_vnd(coins: Money) -> int:
        """Converts Kano-Coin to VND"""
        return coins.amount * CurrencyExchangeService.RATE_COIN_TO_VND

    @staticmethod
    def vnd_to_coin(vnd_amount: int) -> Money:
        """Converts VND to Kano-Coin. Returns Money."""
        coins = int(vnd_amount // CurrencyExchangeService.RATE_COIN_TO_VND)
        return Money(coins)


class CommissionCalculatorService:
    @staticmethod
    def calculate_commission(amount: Money, rate: float) -> int:
        """Calculates system commission using standard mathematical rounding (round())"""
        return int(round(amount.amount * rate))
