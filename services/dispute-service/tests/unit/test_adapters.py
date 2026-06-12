from unittest.mock import MagicMock
from sqlalchemy.ext.asyncio import AsyncSession
from internal.bootstrap import bootstrap_services, settings
from internal.infrastructure.adapters import (
    MockFinanceAdapter,
    MockInteractionAdapter,
    gRPCFinanceAdapter,
    gRPCInteractionAdapter,
)


def test_bootstrap_services_with_mocks():
    """
    Test that when USE_MOCKS is True, Mock adapters are instantiated.
    """
    # Force USE_MOCKS to True
    original_use_mocks = settings.USE_MOCKS
    settings.USE_MOCKS = True

    try:
        mock_db_session = MagicMock(spec=AsyncSession)
        cmd_service, query_service = bootstrap_services(mock_db_session)

        # Verify refund saga orchestrator uses mock adapters
        refund_orch = cmd_service.refund_saga_orchestrator
        assert isinstance(refund_orch.finance_port, MockFinanceAdapter)
        assert isinstance(refund_orch.interaction_port, MockInteractionAdapter)

        # Verify payout saga orchestrator uses mock adapters
        payout_orch = cmd_service.payout_saga_orchestrator
        assert isinstance(payout_orch.finance_port, MockFinanceAdapter)
        assert isinstance(payout_orch.interaction_port, MockInteractionAdapter)

    finally:
        # Restore settings
        settings.USE_MOCKS = original_use_mocks


def test_bootstrap_services_with_grpc():
    """
    Test that when USE_MOCKS is False, gRPC adapters are instantiated with the configured addresses.
    """
    # Force USE_MOCKS to False and set custom addresses
    original_use_mocks = settings.USE_MOCKS
    original_finance_addr = settings.FINANCE_SERVICE_ADDR
    original_interaction_addr = settings.INTERACTION_SERVICE_ADDR

    settings.USE_MOCKS = False
    settings.FINANCE_SERVICE_ADDR = "finance-service:50052"
    settings.INTERACTION_SERVICE_ADDR = "interaction-service:50053"

    try:
        mock_db_session = MagicMock(spec=AsyncSession)
        cmd_service, query_service = bootstrap_services(mock_db_session)

        # Verify refund saga orchestrator uses gRPC adapters
        refund_orch = cmd_service.refund_saga_orchestrator
        assert isinstance(refund_orch.finance_port, gRPCFinanceAdapter)
        assert isinstance(refund_orch.interaction_port, gRPCInteractionAdapter)
        assert refund_orch.finance_port.address == "finance-service:50052"
        assert refund_orch.interaction_port.address == "interaction-service:50053"

        # Verify payout saga orchestrator uses gRPC adapters
        payout_orch = cmd_service.payout_saga_orchestrator
        assert isinstance(payout_orch.finance_port, gRPCFinanceAdapter)
        assert isinstance(payout_orch.interaction_port, gRPCInteractionAdapter)
        assert payout_orch.finance_port.address == "finance-service:50052"
        assert payout_orch.interaction_port.address == "interaction-service:50053"

    finally:
        # Restore settings
        settings.USE_MOCKS = original_use_mocks
        settings.FINANCE_SERVICE_ADDR = original_finance_addr
        settings.INTERACTION_SERVICE_ADDR = original_interaction_addr
