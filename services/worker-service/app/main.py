"""Worker Service — processes async tasks via Redis queue."""

import os
from contextlib import asynccontextmanager

from fastapi import FastAPI
from prometheus_client import make_asgi_app

from app.routes import router
from app.worker import TaskWorker


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Start background worker on startup, stop on shutdown."""
    worker = TaskWorker.get_instance()
    await worker.start()
    yield
    await worker.stop()


app = FastAPI(
    title="Worker Service",
    version="1.0.0",
    lifespan=lifespan,
)

# Routes
app.include_router(router)

# Prometheus metrics endpoint
metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)


@app.get("/health")
async def health():
    return {"status": "ok", "service": "worker-service"}


@app.get("/ready")
async def ready():
    worker = TaskWorker.get_instance()
    redis_ok = await worker.check_redis()
    return {
        "status": "ready" if redis_ok else "not_ready",
        "redis": "connected" if redis_ok else "disconnected",
    }
