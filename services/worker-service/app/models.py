"""Data models for the worker service."""

from datetime import datetime
from enum import Enum
from typing import Optional

from pydantic import BaseModel, Field


class TaskStatus(str, Enum):
    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"


class TaskCreate(BaseModel):
    name: str = Field(..., min_length=1, max_length=255)
    payload: dict = Field(default_factory=dict)
    priority: int = Field(default=0, ge=0, le=10)


class Task(BaseModel):
    id: str
    name: str
    payload: dict
    priority: int
    status: TaskStatus
    result: Optional[dict] = None
    created_at: datetime
    updated_at: datetime
    completed_at: Optional[datetime] = None
    created_by: str = ""
