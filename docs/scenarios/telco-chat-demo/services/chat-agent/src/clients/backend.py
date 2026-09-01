"""Thin async HTTP client over the two backend services.

Every call forwards ``X-Actor-Role`` / ``X-Actor-Id`` headers so the backend
services can log which caller (customer or employee) triggered the request.
Errors are raised as ``BackendError`` with a short, model-safe message —
callers (the tool functions in ``src.agent.tools``) catch this and turn it
into a ``{"error": "..."}`` result rather than letting it crash the stream.
"""

from __future__ import annotations

import logging
from typing import Any

import httpx

logger = logging.getLogger(__name__)

_TIMEOUT_SECONDS = 15.0


class BackendError(Exception):
    """A backend call failed in a way a tool should report, not crash on."""

    def __init__(self, message: str, *, status_code: int | None = None):
        super().__init__(message)
        self.status_code = status_code


class BackendClient:
    """Wraps subscription-service and network-ops-service over httpx."""

    def __init__(self, subscription_base_url: str, network_ops_base_url: str):
        self._subscription_base = subscription_base_url.rstrip("/")
        self._network_ops_base = network_ops_base_url.rstrip("/")
        self._client = httpx.AsyncClient(timeout=_TIMEOUT_SECONDS)

    async def close(self) -> None:
        await self._client.aclose()

    # ── low-level request helper ───────────────────────────────────

    async def _request(
        self,
        method: str,
        base_url: str,
        path: str,
        *,
        headers: dict[str, str],
        params: dict[str, Any] | None = None,
        json_body: dict[str, Any] | None = None,
        tolerate_404: bool = False,
    ) -> Any:
        if not base_url:
            raise BackendError(
                f"cannot call {path}: backend base URL is not configured"
            )
        url = f"{base_url}{path}"
        try:
            resp = await self._client.request(
                method,
                url,
                params=params,
                json=json_body,
                headers=headers,
            )
        except httpx.HTTPError as e:
            raise BackendError(f"{method} {url} failed: {e}") from e

        if resp.status_code == 404 and tolerate_404:
            return None

        if resp.is_error:
            snippet = resp.text[:300] if resp.text else ""
            raise BackendError(
                f"{method} {url} returned HTTP {resp.status_code}: {snippet}",
                status_code=resp.status_code,
            )

        if not resp.content:
            return {}
        try:
            return resp.json()
        except ValueError:
            return {"raw": resp.text}

    # ── subscription-service ───────────────────────────────────────

    async def subscription_get(
        self,
        path: str,
        *,
        headers: dict[str, str],
        params: dict[str, Any] | None = None,
        tolerate_404: bool = False,
    ) -> Any:
        return await self._request(
            "GET",
            self._subscription_base,
            path,
            headers=headers,
            params=params,
            tolerate_404=tolerate_404,
        )

    async def subscription_post(
        self, path: str, *, headers: dict[str, str], json_body: dict[str, Any] | None = None
    ) -> Any:
        return await self._request(
            "POST", self._subscription_base, path, headers=headers, json_body=json_body
        )

    async def subscription_put(
        self, path: str, *, headers: dict[str, str], json_body: dict[str, Any] | None = None
    ) -> Any:
        return await self._request(
            "PUT", self._subscription_base, path, headers=headers, json_body=json_body
        )

    async def subscription_delete(
        self, path: str, *, headers: dict[str, str]
    ) -> Any:
        return await self._request(
            "DELETE", self._subscription_base, path, headers=headers
        )

    # ── network-ops-service ────────────────────────────────────────

    async def network_ops_get(
        self,
        path: str,
        *,
        headers: dict[str, str],
        params: dict[str, Any] | None = None,
        tolerate_404: bool = False,
    ) -> Any:
        return await self._request(
            "GET",
            self._network_ops_base,
            path,
            headers=headers,
            params=params,
            tolerate_404=tolerate_404,
        )

    async def network_ops_post(
        self, path: str, *, headers: dict[str, str], json_body: dict[str, Any] | None = None
    ) -> Any:
        return await self._request(
            "POST", self._network_ops_base, path, headers=headers, json_body=json_body
        )

    async def network_ops_patch(
        self, path: str, *, headers: dict[str, str], json_body: dict[str, Any] | None = None
    ) -> Any:
        return await self._request(
            "PATCH", self._network_ops_base, path, headers=headers, json_body=json_body
        )
