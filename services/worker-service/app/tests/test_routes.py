"""Tests for the worker service API."""

from unittest.mock import AsyncMock, patch

import pytest
from fastapi.testclient import TestClient

from app.main import app
from app.models import Task, TaskStatus
from datetime import datetime, timezone


@pytest.fixture
def client():
    return TestClient(app)


@pytest.fixture
def mock_worker():
    with patch("app.routes.TaskWorker") as mock:
        instance = AsyncMock()
        mock.get_instance.return_value = instance
        yield instance


def test_health(client):
    resp = client.get("/health")
    assert resp.status_code == 200
    assert resp.json()["status"] == "ok"


def test_create_task(client, mock_worker):
    mock_worker.enqueue = AsyncMock()

    resp = client.post(
        "/tasks",
        json={"name": "echo", "payload": {"msg": "hello"}},
        headers={"X-User-ID": "alice"},
    )
    assert resp.status_code == 201
    data = resp.json()
    assert data["name"] == "echo"
    assert data["status"] == "pending"
    assert data["created_by"] == "alice"


def test_create_task_validation(client, mock_worker):
    resp = client.post("/tasks", json={"name": "", "payload": {}})
    assert resp.status_code == 422


def test_list_tasks(client, mock_worker):
    now = datetime.now(timezone.utc)
    mock_worker.list_tasks = AsyncMock(return_value=[
        Task(
            id="1", name="test", payload={}, priority=0,
            status=TaskStatus.COMPLETED, created_at=now, updated_at=now,
        ),
    ])

    resp = client.get("/tasks")
    assert resp.status_code == 200
    assert len(resp.json()) == 1


def test_get_task_not_found(client, mock_worker):
    mock_worker.get_task = AsyncMock(return_value=None)

    resp = client.get("/tasks/nonexistent")
    assert resp.status_code == 404
