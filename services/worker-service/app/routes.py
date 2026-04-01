"""API routes for task management."""

import uuid
from datetime import datetime, timezone

from fastapi import APIRouter, Header, HTTPException

from app.models import Task, TaskCreate, TaskStatus
from app.worker import TaskWorker

router = APIRouter(prefix="/tasks", tags=["tasks"])


@router.post("", status_code=201)
async def create_task(
    task: TaskCreate,
    x_user_id: str = Header(default="anonymous"),
) -> Task:
    """Create a new task and enqueue it for processing."""
    now = datetime.now(timezone.utc)
    new_task = Task(
        id=str(uuid.uuid4()),
        name=task.name,
        payload=task.payload,
        priority=task.priority,
        status=TaskStatus.PENDING,
        created_at=now,
        updated_at=now,
        created_by=x_user_id,
    )

    worker = TaskWorker.get_instance()
    await worker.enqueue(new_task)

    return new_task


@router.get("")
async def list_tasks(
    status: TaskStatus | None = None,
    limit: int = 50,
) -> list[Task]:
    """List tasks, optionally filtered by status."""
    worker = TaskWorker.get_instance()
    tasks = await worker.list_tasks()

    if status:
        tasks = [t for t in tasks if t.status == status]

    return tasks[:limit]


@router.get("/{task_id}")
async def get_task(task_id: str) -> Task:
    """Get a specific task by ID."""
    worker = TaskWorker.get_instance()
    task = await worker.get_task(task_id)

    if not task:
        raise HTTPException(status_code=404, detail="Task not found")

    return task
