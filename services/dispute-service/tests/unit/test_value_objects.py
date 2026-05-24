import pytest
from internal.domain.vo import DisputeReason, Resolution
from internal.domain.errors import InvalidDisputeReasonError, InvalidResolutionError


def test_dispute_reason_valid():
    reasons = ["NO_SHOW", "MISCONDUCT", "FRAUD", "OTHER"]
    for r in reasons:
        vo = DisputeReason(r)
        assert vo.value == r
        assert str(vo) == r


def test_dispute_reason_invalid():
    with pytest.raises(InvalidDisputeReasonError):
        DisputeReason("BAD_SERVICE")


def test_resolution_valid():
    resolutions = ["REFUND_CLIENT", "PAYOUT_COMPANION", "REJECT"]
    for res in resolutions:
        vo = Resolution(res)
        assert vo.value == res
        assert str(vo) == res
        
        if res == "REFUND_CLIENT":
            assert vo.is_refund is True
            assert vo.is_payout is False
            assert vo.is_reject is False
        elif res == "PAYOUT_COMPANION":
            assert vo.is_refund is False
            assert vo.is_payout is True
            assert vo.is_reject is False
        elif res == "REJECT":
            assert vo.is_refund is False
            assert vo.is_payout is False
            assert vo.is_reject is True


def test_resolution_invalid():
    with pytest.raises(InvalidResolutionError):
        Resolution("DO_NOTHING")
