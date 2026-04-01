"""Background task worker using Redis as queue."""

import asyncio
import json
import logging
import os
from datetime import datetime, timezone
from typing import Optional

import redis.asyncio as redis
from prometheus_client import Counter, Histogram

from app.models import Task, TaskStatus

logger = logging.getLogger(__name__)

# Prometheus metrics
tasks_processed = Counter(
    "worker_tasks_processed_total",
    "Total tasks processed",
    ["status"],
)

task_duration = Histogram(
    "worker_task_duration_seconds",
    "Task processing duration",
    buckets=[0.1, 0.5, 1, 2, 5, 10, 30, 60],
)

REDIS_URL = os.getenv("REDIS_URL", "redis://redis:6379/0")
QUEUE_KEY = "task_queue"
TASKS_KEY = "tasks"


class TaskWorker:
    _instance: Optional["TaskWorker"] = None

    def __init__(self):
        self.redis: Optional[redis.Redis] = None
        self._running = False
        self._worker_task: Optional[asyncio.Task] = None

    @classmethod
    def get_instance(cls) -> "TaskWorker":
        if cls._instance is None:
            cls._instance = cls()
        return cls._instance

    async def start(self):
        """Connect to Redis and start the background worker loop."""
        self.redis = redis.from_url(REDIS_URL, decode_responses=True)
        self._running = True
        self._worker_task = asyncio.create_task(self._process_loop())
        logger.info("Worker started, connected to Redis")

    async def stop(self):
        """Stop the worker and close Redis connection."""
        self._running = False
        if self._worker_task:
            self._worker_task.cancel()
            try:
                await self._worker_task
            except asyncio.CancelledError:
                pass
        if self.redis:
            await self.redis.close()
        logger.info("Worker stopped")

    async def check_redis(self) -> bool:
        """Check Redis connectivity."""
        try:
            if self.redis:
                await self.redis.ping()
                return True
        except Exception:
            pass
        return False

    async def enqueue(self, task: Task):
        """Add a task to the Redis queue and store it."""
        task_json = task.model_dump_json()
        await self.redis.hset(TASKS_KEY, task.id, task_json)
        await self.redis.lpush(QUEUE_KEY, task.id)
        logger.info(f"Enqueued task {task.id}: {task.name}")

    async def get_task(self, task_id: str) -> Optional[Task]:
        """Retrieve a task by ID."""
        data = await self.redis.hget(TASKS_KEY, task_id)
        if not data:
            return None
        return Task.model_validate_json(data)

    async def list_tasks(self) -> list[Task]:
        """List all tasks."""
        all_data = await self.redis.hgetall(TASKS_KEY)
        tasks = []
        for data in all_data.values():
            tasks.append(Task.model_validate_json(data))
        tasks.sort(key=lambda t: t.created_at, reverse=True)
        return tasks

    async def _process_loop(self):
        """Main worker loop — poll Redis queue and process tasks."""
        logger.info("Worker loop started")
        while self._running:
            try:
                # Blocking pop with 1s timeout
                result = await self.redis.brpop(QUEUE_KEY, timeout=1)
                if result is None:
                    continue

                _, task_id = result
                await self._process_task(task_id)

            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"Worker loop error: {e}")
                await asyncio.sleep(1)

    async def _process_task(self, task_id: str):
        """Process a single task."""
        task = await self.get_task(task_id)
        if not task:
            logger.warning(f"Task {task_id} not found, skipping")
            return

        logger.info(f"Processing task {task_id}: {task.name}")

        # Update status to processing
        task.status = TaskStatus.PROCESSING
        task.updated_at = datetime.now(timezone.utc)
        await self.redis.hset(TASKS_KEY, task.id, task.model_dump_json())

        start_time = asyncio.get_event_loop().time()

        try:
            # Simulate task processing
            result = await self._execute_task(task)

            task.status = TaskStatus.COMPLETED
            task.result = result
            task.completed_at = datetime.now(timezone.utc)
            tasks_processed.labels(status="completed").inc()

        except Exception as e:
            logger.error(f"Task {task_id} failed: {e}")
            task.status = TaskStatus.FAILED
            task.result = {"error": str(e)}
            tasks_processed.labels(status="failed").inc()

        finally:
            duration = asyncio.get_event_loop().time() - start_time
            task_duration.observe(duration)
            task.updated_at = datetime.now(timezone.utc)
            await self.redis.hset(TASKS_KEY, task.id, task.model_dump_json())

        logger.info(f"Task {task_id} {task.status.value} in {duration:.2f}s")

    async def _execute_task(self, task: Task) -> dict:
        """Execute task logic based on task name."""
        # Simulate different task types
        match task.name:
            case "echo":
                return {"echo": task.payload}
            case "compute":
                await asyncio.sleep(2)  # Simulate computation
                return {"result": "computed", "input": task.payload}
            case "notify":
                await asyncio.sleep(0.5)
                return {"notified": True, "channel": task.payload.get("channel", "default")}
            case _:
                await asyncio.sleep(1)
                return {"processed": True, "task_name": task.name}
