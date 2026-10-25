"""Хранение моделей и статусов задач обучения.

Модель Prophet сериализуется через ``model_to_json`` — это официальный
способ, pickle для Prophet не рекомендуется. Статусы задач лежат рядом
обычным JSON, чтобы бэкенд мог опрашивать их между рестартами сервиса.
"""

from __future__ import annotations

import json
import threading
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from prophet.serialize import model_from_json, model_to_json

from . import config


class Registry:
    def __init__(self, root: Path = config.ARTIFACTS_DIR) -> None:
        self.root = root
        self.models_dir = root / "models"
        self.jobs_dir = root / "jobs"
        self.uploads_dir = root / "uploads"
        for directory in (self.models_dir, self.jobs_dir, self.uploads_dir):
            directory.mkdir(parents=True, exist_ok=True)
        self._lock = threading.Lock()

    # --- модели ---------------------------------------------------------

    def save_model(self, company_id: str, model: Any, meta: dict[str, Any]) -> None:
        company_dir = self.models_dir / company_id
        company_dir.mkdir(parents=True, exist_ok=True)
        (company_dir / "model.json").write_text(model_to_json(model), encoding="utf-8")
        (company_dir / "meta.json").write_text(
            json.dumps(meta, ensure_ascii=False, indent=2), encoding="utf-8"
        )

    def load_model(self, company_id: str) -> tuple[Any, dict[str, Any]]:
        company_dir = self.models_dir / company_id
        model_path = company_dir / "model.json"
        if not model_path.exists():
            raise FileNotFoundError(company_id)
        model = model_from_json(model_path.read_text(encoding="utf-8"))
        meta = json.loads((company_dir / "meta.json").read_text(encoding="utf-8"))
        return model, meta

    def has_model(self, company_id: str) -> bool:
        return (self.models_dir / company_id / "model.json").exists()

    def list_models(self) -> list[dict[str, Any]]:
        result = []
        for company_dir in sorted(p for p in self.models_dir.iterdir() if p.is_dir()):
            meta_path = company_dir / "meta.json"
            if meta_path.exists():
                meta = json.loads(meta_path.read_text(encoding="utf-8"))
                result.append({"company_id": company_dir.name, **meta})
        return result

    def delete_model(self, company_id: str) -> bool:
        company_dir = self.models_dir / company_id
        if not company_dir.exists():
            return False
        for path in company_dir.iterdir():
            path.unlink()
        company_dir.rmdir()
        return True

    # --- задачи ---------------------------------------------------------

    def create_job(self, job_id: str, company_id: str) -> dict[str, Any]:
        job = {
            "job_id": job_id,
            "company_id": company_id,
            "status": "queued",
            "created_at": _now(),
            "started_at": None,
            "finished_at": None,
            "error": None,
            "metrics": None,
            "params": None,
        }
        self._write_job(job)
        return job

    def update_job(self, job_id: str, **changes: Any) -> dict[str, Any]:
        with self._lock:
            job = self._read_job(job_id)
            job.update(changes)
            self._write_job(job)
            return job

    def get_job(self, job_id: str) -> dict[str, Any]:
        return self._read_job(job_id)

    def fail_unfinished_jobs(self, reason: str) -> int:
        """Помечает задачи, оборванные рестартом сервиса, как failed.

        Фоновые задачи живут в памяти процесса: после перезапуска статус
        ``running`` никогда не сменился бы, и бэкенд ждал бы вечно.
        """
        count = 0
        for path in self.jobs_dir.glob("*.json"):
            try:
                job = json.loads(path.read_text(encoding="utf-8"))
            except (json.JSONDecodeError, OSError):
                continue
            if job.get("status") in ("queued", "running"):
                job.update(status="failed", finished_at=_now(), error=reason)
                path.write_text(
                    json.dumps(job, ensure_ascii=False, indent=2), encoding="utf-8"
                )
                count += 1
        return count

    def _job_path(self, job_id: str) -> Path:
        return self.jobs_dir / f"{job_id}.json"

    def _read_job(self, job_id: str) -> dict[str, Any]:
        path = self._job_path(job_id)
        if not path.exists():
            raise FileNotFoundError(job_id)
        return json.loads(path.read_text(encoding="utf-8"))

    def _write_job(self, job: dict[str, Any]) -> None:
        self._job_path(job["job_id"]).write_text(
            json.dumps(job, ensure_ascii=False, indent=2), encoding="utf-8"
        )


def _now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


registry = Registry()
