FROM python:3.12-slim AS builder

RUN pip install --no-cache-dir uv

WORKDIR /app

COPY pyproject.toml README.md ./
COPY src ./src

# No uv.lock is vendored — resolve directly against pyproject.toml into an
# isolated venv (kept separate from the runtime stage's Python so only the
# venv + its resolved deps get copied forward).
RUN uv venv /opt/venv \
 && uv pip install --python /opt/venv/bin/python --no-cache .

FROM python:3.12-slim

RUN groupadd --system --gid 10001 chatagent \
 && useradd --system --uid 10001 --gid chatagent --no-create-home chatagent

COPY --from=builder --chown=chatagent:chatagent /opt/venv /opt/venv
WORKDIR /app
COPY --from=builder --chown=chatagent:chatagent /app/src ./src

ENV PATH="/opt/venv/bin:$PATH" \
    PYTHONUNBUFFERED=1

USER chatagent

EXPOSE 8080

CMD ["uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8080"]
