"""Runs one chat turn and yields NDJSON lines per the /agent/stream contract.

Uses ``agent.astream_events(..., version="v2")`` rather than manually
reassembling ``stream_mode="messages"`` chunks: the v2 event stream hands us
``on_chat_model_stream`` for incremental text, and — critically —
``on_tool_start`` / ``on_tool_end`` with the tool's fully-parsed input dict
and real return value, so ``tool_call``/``tool_result`` events never need to
reconstruct partial tool-call JSON by hand.
"""

from __future__ import annotations

import json
import logging
from collections.abc import AsyncIterator
from typing import Any

from langchain.agents import create_agent
from langchain_core.messages import AIMessage, BaseMessage, HumanMessage

from src.agent.context import ToolContext
from src.agent.prompt import build_system_prompt
from src.agent.tools import build_tools, compute_audit, compute_target_customer_id
from src.clients import BackendClient, get_model
from src.models import AgentStreamRequest

logger = logging.getLogger(__name__)


def _emit(obj: dict[str, Any]) -> str:
    return json.dumps(obj, separators=(",", ":")) + "\n"


def _build_messages(req: AgentStreamRequest) -> list[BaseMessage]:
    messages: list[BaseMessage] = []
    for turn in req.history:
        if turn.role == "assistant":
            messages.append(AIMessage(content=turn.content))
        else:
            messages.append(HumanMessage(content=turn.content))
    messages.append(HumanMessage(content=req.message))
    return messages


def _extract_text(chunk: Any) -> str:
    """Pull incremental text out of an AIMessageChunk, if any."""
    content = getattr(chunk, "content", "")
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        parts = []
        for block in content:
            if isinstance(block, dict) and block.get("type") == "text":
                parts.append(block.get("text", ""))
            elif isinstance(block, str):
                parts.append(block)
        return "".join(parts)
    return ""


def _extract_tool_result(output: Any) -> Any:
    """Turn a ToolMessage (or raw return value) into a JSON-able result."""
    content = getattr(output, "content", output)
    if isinstance(content, str):
        try:
            return json.loads(content)
        except ValueError:
            return {"value": content}
    return content


async def stream_chat(
    req: AgentStreamRequest,
    backend: BackendClient,
) -> AsyncIterator[str]:
    ctx = ToolContext(
        role=req.role,
        actor_id=req.actor_id,
        target_customer_id=req.target_customer_id,
    )

    logger.info(
        "chat turn start conversation_id=%s role=%s actor_id=%s target_customer_id=%s",
        req.conversation_id,
        ctx.role,
        ctx.actor_id,
        ctx.target_customer_id,
    )

    try:
        tools = build_tools(ctx, backend)
        model = get_model()
        agent = create_agent(
            model=model,
            tools=tools,
            system_prompt=build_system_prompt(ctx),
        )
    except Exception as e:
        logger.exception("failed to initialize agent")
        yield _emit({"type": "error", "message": f"Failed to initialize the assistant: {e}"})
        return

    messages = _build_messages(req)

    # Tracks whether the model narrated the most recent tool error itself —
    # if a tool_result carries an "error" and no token follows it before the
    # turn ends, we inject one short explanatory token as a safety net.
    pending_tool_error: str | None = None
    narrated_since_error = True

    try:
        async for event in agent.astream_events({"messages": messages}, version="v2"):
            kind = event.get("event")
            data = event.get("data", {})

            if kind == "on_chat_model_stream":
                text = _extract_text(data.get("chunk"))
                if text:
                    if pending_tool_error is not None:
                        narrated_since_error = True
                    yield _emit({"type": "token", "content": text})

            elif kind == "on_tool_start":
                name = event.get("name", "")
                args = data.get("input")
                if not isinstance(args, dict):
                    args = {"input": args}
                yield _emit(
                    {
                        "type": "tool_call",
                        "name": name,
                        "args": args,
                        "audit": compute_audit(name, args),
                        "targetCustomerId": compute_target_customer_id(name, args, ctx),
                    }
                )

            elif kind == "on_tool_end":
                name = event.get("name", "")
                result = _extract_tool_result(data.get("output"))
                yield _emit({"type": "tool_result", "name": name, "result": result})
                if isinstance(result, dict) and result.get("error"):
                    pending_tool_error = str(result["error"])
                    narrated_since_error = False

            elif kind == "on_tool_error":
                name = event.get("name", "")
                err = str(data.get("error") or "tool failed")
                yield _emit({"type": "tool_result", "name": name, "result": {"error": err}})
                pending_tool_error = err
                narrated_since_error = False

        if pending_tool_error is not None and not narrated_since_error:
            yield _emit(
                {
                    "type": "token",
                    "content": (
                        f"\n\n(I ran into a problem completing that: {pending_tool_error}. "
                        "Let me know if you'd like me to try again.)"
                    ),
                }
            )

        yield _emit({"type": "done"})

    except Exception as e:
        logger.exception("chat stream failed conversation_id=%s", req.conversation_id)
        yield _emit({"type": "error", "message": f"The assistant hit an unrecoverable error: {e}"})
