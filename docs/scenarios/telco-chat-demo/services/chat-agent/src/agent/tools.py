"""The full tool catalog, and role-scoped filtering.

Every ``make_*`` function builds one LangChain ``StructuredTool`` closed over
the per-request ``ToolContext`` and ``BackendClient`` — this is how a
customer-role tool implicitly scopes itself to ``ctx.target_customer_id``
without ever accepting a customer id as a model-visible argument.

``build_tools`` is the single entry point: it returns only the tools for
``ctx.role``, so a customer-role request never even has the employee tools
bound to the model (not just prompt-instructed not to use them).
"""

from __future__ import annotations

import logging
from datetime import UTC, datetime
from typing import Any, Literal

from langchain_core.tools import BaseTool, StructuredTool
from pydantic import BaseModel, Field

from src.agent.context import ToolContext
from src.clients.backend import BackendClient, BackendError

logger = logging.getLogger(__name__)


def _today() -> str:
    return datetime.now(UTC).strftime("%Y-%m-%d")


async def _safe(coro_call, *, tool_name: str) -> dict[str, Any]:
    """Run a tool body, turning any failure into a result dict.

    Never raises — a failed backend call must surface to the model as a
    ToolMessage it can read and narrate, not crash the agent stream.
    """
    try:
        return await coro_call()
    except BackendError as e:
        logger.warning("tool %s failed: %s", tool_name, e)
        return {"error": str(e)}
    except Exception as e:  # noqa: BLE001 — last-resort guard, see docstring
        logger.exception("tool %s failed unexpectedly", tool_name)
        return {"error": f"unexpected error calling {tool_name}: {e}"}


# ── customer tools ──────────────────────────────────────────────────
# All implicitly scoped to ctx.target_customer_id — never accept a customer
# id argument, so the model has no way to point them at another account.


def make_get_my_subscription(ctx: ToolContext, backend: BackendClient) -> BaseTool:
    async def _run() -> dict[str, Any]:
        async def call():
            return await backend.subscription_get(
                f"/customers/{ctx.target_customer_id}/subscription",
                headers=ctx.headers,
            )

        return await _safe(call, tool_name="get_my_subscription")

    return StructuredTool.from_function(
        coroutine=_run,
        name="get_my_subscription",
        description="Get the caller's own current subscription plan, status, and billing details.",
    )


def make_list_available_plans(ctx: ToolContext, backend: BackendClient) -> BaseTool:
    async def _run() -> dict[str, Any]:
        async def call():
            return await backend.subscription_get("/plans", headers=ctx.headers)

        return await _safe(call, tool_name="list_available_plans")

    return StructuredTool.from_function(
        coroutine=_run,
        name="list_available_plans",
        description="List all subscription plans currently available to switch to.",
    )


class ChangeMySubscriptionArgs(BaseModel):
    plan_id: str = Field(description="The id of the plan to switch the caller's subscription to.")


def make_change_my_subscription(ctx: ToolContext, backend: BackendClient) -> BaseTool:
    async def _run(plan_id: str) -> dict[str, Any]:
        async def call():
            return await backend.subscription_post(
                f"/customers/{ctx.target_customer_id}/subscription",
                headers=ctx.headers,
                json_body={"planId": plan_id},
            )

        return await _safe(call, tool_name="change_my_subscription")

    return StructuredTool.from_function(
        coroutine=_run,
        name="change_my_subscription",
        description=(
            "Change the caller's own subscription to a different plan. "
            "This mutates the account — confirm the plan with the caller first."
        ),
        args_schema=ChangeMySubscriptionArgs,
    )


class GetMyUsageArgs(BaseModel):
    date: str | None = Field(
        default=None,
        description=(
            "Date to fetch usage for, as YYYY-MM-DD. Omit to get today's usage — "
            "never guess a date yourself, leave this unset instead."
        ),
    )


def make_get_my_usage(ctx: ToolContext, backend: BackendClient) -> BaseTool:
    async def _run(date: str | None = None) -> dict[str, Any]:
        async def call():
            return await backend.network_ops_get(
                f"/customers/{ctx.target_customer_id}/usage",
                headers=ctx.headers,
                params={"date": date or _today()},
            )

        return await _safe(call, tool_name="get_my_usage")

    return StructuredTool.from_function(
        coroutine=_run,
        name="get_my_usage",
        description="Get the caller's own network data usage for a given day (defaults to today).",
        args_schema=GetMyUsageArgs,
    )


class ReportServiceIssueArgs(BaseModel):
    category: str = Field(
        description="Short category for the issue, e.g. 'no_signal', 'slow_data', 'billing'."
    )
    description: str = Field(description="Free-text description of the issue, from the caller.")


def make_report_service_issue(ctx: ToolContext, backend: BackendClient) -> BaseTool:
    async def _run(category: str, description: str) -> dict[str, Any]:
        async def call():
            return await backend.network_ops_post(
                "/reports",
                headers=ctx.headers,
                json_body={
                    "customerId": ctx.target_customer_id,
                    "category": category,
                    "description": description,
                },
            )

        return await _safe(call, tool_name="report_service_issue")

    return StructuredTool.from_function(
        coroutine=_run,
        name="report_service_issue",
        description="File a new service issue report on the caller's own account.",
        args_schema=ReportServiceIssueArgs,
    )


# ── employee tools ──────────────────────────────────────────────────
# Require an explicit customer id argument — never default to the employee's
# own actor id.


class LookupCustomerArgs(BaseModel):
    query: str = Field(description="Search text — name, email, phone, or customer id fragment.")


def make_lookup_customer(ctx: ToolContext, backend: BackendClient) -> BaseTool:
    async def _run(query: str) -> dict[str, Any]:
        async def call():
            return await backend.subscription_get(
                "/customers", headers=ctx.headers, params={"search": query}
            )

        return await _safe(call, tool_name="lookup_customer")

    return StructuredTool.from_function(
        coroutine=_run,
        name="lookup_customer",
        description="Search for customers by name, email, phone, or id fragment.",
        args_schema=LookupCustomerArgs,
    )


class GetCustomerAccountArgs(BaseModel):
    customer_id: str = Field(description="The customer id to look up.")


def make_get_customer_account(ctx: ToolContext, backend: BackendClient) -> BaseTool:
    async def _run(customer_id: str) -> dict[str, Any]:
        async def call():
            profile = await backend.subscription_get(
                f"/customers/{customer_id}", headers=ctx.headers
            )
            subscription = await backend.subscription_get(
                f"/customers/{customer_id}/subscription", headers=ctx.headers
            )
            usage = await backend.network_ops_get(
                f"/customers/{customer_id}/usage",
                headers=ctx.headers,
                params={"date": _today()},
                tolerate_404=True,
            )
            reports = await backend.network_ops_get(
                "/reports", headers=ctx.headers, params={"customerId": customer_id}
            )
            account: dict[str, Any] = {
                "profile": profile,
                "subscription": subscription,
                "reports": reports,
            }
            # No usage record for today is fine — omit the field entirely
            # rather than surfacing the 404 as a merge error.
            if usage is not None:
                account["usage"] = usage
            return account

        return await _safe(call, tool_name="get_customer_account")

    return StructuredTool.from_function(
        coroutine=_run,
        name="get_customer_account",
        description=(
            "Get a merged view of one customer's account: profile, subscription, "
            "today's usage (if any), and open service reports."
        ),
        args_schema=GetCustomerAccountArgs,
    )


class ChangeCustomerSubscriptionArgs(BaseModel):
    customer_id: str = Field(description="The customer id whose subscription to change.")
    plan_id: str = Field(description="The id of the plan to switch the customer to.")


def make_change_customer_subscription(ctx: ToolContext, backend: BackendClient) -> BaseTool:
    async def _run(customer_id: str, plan_id: str) -> dict[str, Any]:
        async def call():
            return await backend.subscription_post(
                f"/customers/{customer_id}/subscription",
                headers=ctx.headers,
                json_body={"planId": plan_id},
            )

        return await _safe(call, tool_name="change_customer_subscription")

    return StructuredTool.from_function(
        coroutine=_run,
        name="change_customer_subscription",
        description=(
            "Change a specific customer's subscription to a different plan. "
            "This mutates the customer's account — confirm before calling."
        ),
        args_schema=ChangeCustomerSubscriptionArgs,
    )


class ManagePlanCatalogArgs(BaseModel):
    action: Literal["list", "create", "update", "deactivate"] = Field(
        description="Which plan-catalog operation to perform."
    )
    plan_id: str | None = Field(
        default=None, description="Required for update/deactivate; the plan id to act on."
    )
    name: str | None = Field(default=None, description="Plan name (create/update).")
    data_gb: int | None = Field(
        default=None, description="Monthly data allowance in GB (create/update)."
    )
    price_cents: int | None = Field(
        default=None, description="Monthly price in cents (create/update)."
    )


def make_manage_plan_catalog(ctx: ToolContext, backend: BackendClient) -> BaseTool:
    async def _run(
        action: Literal["list", "create", "update", "deactivate"],
        plan_id: str | None = None,
        name: str | None = None,
        data_gb: int | None = None,
        price_cents: int | None = None,
    ) -> dict[str, Any]:
        async def call():
            if action == "list":
                return await backend.subscription_get("/plans", headers=ctx.headers)

            if action == "create":
                if not plan_id or not name:
                    return {"error": "action='create' requires plan_id and name"}
                body = {
                    "id": plan_id,
                    "name": name,
                    "dataGb": data_gb,
                    "priceCents": price_cents or 0,
                }
                return await backend.subscription_post("/plans", headers=ctx.headers, json_body=body)

            if not plan_id:
                return {"error": f"action={action!r} requires plan_id"}

            if action == "update":
                # subscription-service's PUT /plans/{id} replaces the whole
                # plan (it has no notion of a partial patch) — fetch the
                # current values first and merge in only what the caller
                # actually asked to change, so an update that only touches
                # price doesn't blank out the name or data allowance.
                current = await backend.subscription_get("/plans", headers=ctx.headers)
                existing = next((p for p in current if p.get("id") == plan_id), None)
                if existing is None:
                    return {"error": f"no such plan: {plan_id}"}
                body = {
                    "id": plan_id,
                    "name": name if name is not None else existing.get("name"),
                    "dataGb": data_gb if data_gb is not None else existing.get("dataGb"),
                    "priceCents": (
                        price_cents if price_cents is not None else existing.get("priceCents")
                    ),
                }
                return await backend.subscription_put(
                    f"/plans/{plan_id}", headers=ctx.headers, json_body=body
                )

            # deactivate
            return await backend.subscription_delete(f"/plans/{plan_id}", headers=ctx.headers)

        return await _safe(call, tool_name="manage_plan_catalog")

    return StructuredTool.from_function(
        coroutine=_run,
        name="manage_plan_catalog",
        description=(
            "List, create, update, or deactivate plans in the subscription plan catalog. "
            "'list' is read-only; create/update/deactivate mutate the catalog."
        ),
        args_schema=ManagePlanCatalogArgs,
    )


class ManageServiceReportsArgs(BaseModel):
    action: Literal["list", "update"] = Field(description="Which service-reports operation to perform.")
    customer_id: str | None = Field(
        default=None, description="Filter (list) or identify (update) reports by customer id."
    )
    status_filter: str | None = Field(
        default=None, description="Filter listed reports by status, e.g. 'open', 'resolved'."
    )
    report_id: str | None = Field(default=None, description="Required for update; the report id to change.")
    new_status: str | None = Field(default=None, description="New status to set (update).")
    resolution_notes: str | None = Field(
        default=None, description="Notes explaining the resolution (update)."
    )


def make_manage_service_reports(ctx: ToolContext, backend: BackendClient) -> BaseTool:
    async def _run(
        action: Literal["list", "update"],
        customer_id: str | None = None,
        status_filter: str | None = None,
        report_id: str | None = None,
        new_status: str | None = None,
        resolution_notes: str | None = None,
    ) -> dict[str, Any]:
        async def call():
            if action == "list":
                params = {
                    k: v
                    for k, v in {"customerId": customer_id, "status": status_filter}.items()
                    if v is not None
                }
                return await backend.network_ops_get(
                    "/reports", headers=ctx.headers, params=params or None
                )

            if not report_id:
                return {"error": "action='update' requires report_id"}
            body = {
                k: v
                for k, v in {"status": new_status, "resolutionNotes": resolution_notes}.items()
                if v is not None
            }
            return await backend.network_ops_patch(
                f"/reports/{report_id}", headers=ctx.headers, json_body=body
            )

        return await _safe(call, tool_name="manage_service_reports")

    return StructuredTool.from_function(
        coroutine=_run,
        name="manage_service_reports",
        description=(
            "List service issue reports (optionally filtered by customer/status), or update "
            "one report's status and resolution notes. 'list' is read-only; 'update' mutates."
        ),
        args_schema=ManageServiceReportsArgs,
    )


# ── role-scoped registry ────────────────────────────────────────────

_CUSTOMER_FACTORIES = [
    make_get_my_subscription,
    make_list_available_plans,
    make_change_my_subscription,
    make_get_my_usage,
    make_report_service_issue,
]

_EMPLOYEE_FACTORIES = [
    make_lookup_customer,
    make_get_customer_account,
    make_change_customer_subscription,
    make_manage_plan_catalog,
    make_manage_service_reports,
]

# Static audit (mutating vs. read-only) classification. Tools whose action
# varies per-call (manage_plan_catalog, manage_service_reports) are resolved
# dynamically in ``compute_audit`` instead of appearing here.
_STATIC_AUDIT: dict[str, bool] = {
    "get_my_subscription": False,
    "list_available_plans": False,
    "change_my_subscription": True,
    "get_my_usage": False,
    "report_service_issue": True,
    "lookup_customer": False,
    "get_customer_account": False,
    "change_customer_subscription": True,
}

# Tools whose targetCustomerId comes from their own customer_id argument
# rather than from ctx.target_customer_id.
_CUSTOMER_ID_ARG_TOOLS = frozenset({"get_customer_account", "change_customer_subscription"})


def build_tools(ctx: ToolContext, backend: BackendClient) -> list[BaseTool]:
    """Return only the tools ``ctx.role`` is allowed to see.

    A customer-role request gets exactly ``_CUSTOMER_FACTORIES`` bound to the
    model — the employee tools are never constructed for it, so there is no
    catalog entry the model could even attempt to call.
    """
    factories = _CUSTOMER_FACTORIES if ctx.role == "customer" else _EMPLOYEE_FACTORIES
    return [factory(ctx, backend) for factory in factories]


def compute_audit(tool_name: str, args: dict[str, Any]) -> bool:
    """Whether invoking ``tool_name`` with ``args`` mutates state."""
    if tool_name in ("manage_plan_catalog", "manage_service_reports"):
        return args.get("action") != "list"
    return _STATIC_AUDIT.get(tool_name, True)


def compute_target_customer_id(
    tool_name: str, args: dict[str, Any], ctx: ToolContext
) -> str | None:
    """The customer id a tool call concerned, for the ``tool_call`` event."""
    if tool_name in _CUSTOMER_ID_ARG_TOOLS:
        return args.get("customer_id")
    if tool_name == "manage_service_reports":
        return args.get("customer_id") or ctx.target_customer_id
    return ctx.target_customer_id
