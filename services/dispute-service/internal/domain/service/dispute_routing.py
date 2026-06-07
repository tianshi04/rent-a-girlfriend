from typing import List, Optional
from internal.domain.aggregate import Dispute


class DisputeRoutingService:
    """
    Domain Service to route/assign Disputes to appropriate Admins.
    Initially implements simple assignment logic (e.g., manual routing or round-robin if online list provided).
    """

    def route_dispute(
        self, dispute: Dispute, available_admins: List[str]
    ) -> Optional[str]:
        """
        Assign an admin to the dispute.
        If a list of available admins is provided, returns the first one (simple mock routing).
        Otherwise returns None (needs manual assignment).
        """
        if not available_admins:
            return None
        # Return first available admin for now
        return available_admins[0]
