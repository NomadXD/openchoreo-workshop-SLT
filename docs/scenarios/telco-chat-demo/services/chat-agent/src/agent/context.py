from dataclasses import dataclass


@dataclass(frozen=True)
class ToolContext:
    """Per-request context closed over by every tool factory in ``tools.py``.

    Never exposed to the model as a tool argument — ``target_customer_id`` in
    particular is derived server-side from the trusted request, never
    accepted as model input for the customer-role tools.
    """

    role: str  # "customer" | "employee"
    actor_id: str
    target_customer_id: str | None

    @property
    def headers(self) -> dict[str, str]:
        return {"X-Actor-Role": self.role, "X-Actor-Id": self.actor_id}
