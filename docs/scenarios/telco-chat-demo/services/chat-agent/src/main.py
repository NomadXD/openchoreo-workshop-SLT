import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request
from fastapi.responses import StreamingResponse

from src.agent import stream_chat
from src.clients import BackendClient
from src.config import settings
from src.models import AgentStreamRequest

logging.basicConfig(level=settings.log_level)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Fail-fast config validation: a runtime OpenAI 401 deep inside a stream
    # is a confusing failure mode; refuse to start instead.
    if not settings.openai_api_key:
        raise RuntimeError(
            "OPENAI_API_KEY is not set — refusing to start. "
            "Set the env var and redeploy."
        )
    if not settings.subscription_service_url:
        logger.warning("SUBSCRIPTION_SERVICE_URL is not set — subscription tools will fail")
    if not settings.network_ops_service_url:
        logger.warning("NETWORK_OPS_SERVICE_URL is not set — network-ops tools will fail")

    app.state.backend = BackendClient(
        subscription_base_url=settings.subscription_service_url,
        network_ops_base_url=settings.network_ops_service_url,
    )
    logger.info(
        "chat-agent starting up: model=%s subscription_service=%s network_ops_service=%s",
        settings.openai_model,
        settings.subscription_service_url,
        settings.network_ops_service_url,
    )

    yield

    await app.state.backend.close()


app = FastAPI(
    title="chat-agent",
    lifespan=lifespan,
    docs_url=None,
    redoc_url=None,
    openapi_url=None,
)


@app.get("/healthz")
async def healthz():
    return {"status": "ok"}


@app.post("/agent/stream")
async def agent_stream(request: AgentStreamRequest, http_request: Request):
    logger.info(
        "agent/stream request conversation_id=%s role=%s actor_id=%s",
        request.conversation_id,
        request.role,
        request.actor_id,
    )
    backend: BackendClient = http_request.app.state.backend
    return StreamingResponse(
        stream_chat(request, backend),
        media_type="application/x-ndjson",
        headers={"Cache-Control": "no-cache"},
    )
