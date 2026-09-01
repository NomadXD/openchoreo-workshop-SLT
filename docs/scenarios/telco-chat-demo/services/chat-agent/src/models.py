from typing import Literal

from pydantic import BaseModel, Field


class HistoryMessage(BaseModel):
    role: Literal["user", "assistant"]
    content: str


class AgentStreamRequest(BaseModel):
    """Body for POST /agent/stream.

    Already fully trusted — chat-gateway has done all auth and scoping before
    this service ever sees the request.
    """

    role: Literal["customer", "employee"]
    actor_id: str = Field(alias="actorId")
    # Always == actor_id when role == "customer". May be absent/None for an
    # employee turn that isn't (yet) about a specific customer.
    target_customer_id: str | None = Field(default=None, alias="targetCustomerId")
    # Opaque, used only for our own logging — never persisted.
    conversation_id: str | None = Field(default=None, alias="conversationId")
    message: str
    history: list[HistoryMessage] = Field(default_factory=list)

    model_config = {"populate_by_name": True}
