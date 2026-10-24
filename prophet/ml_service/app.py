"""FastAPI-обёртка над Prophet.

Обучение тяжёлое (Optuna + rolling CV — минуты на компанию), поэтому
``POST /train`` только ставит задачу и сразу возвращает ``job_id``.
Бэкенд забирает результат через ``GET /jobs/{job_id}``, а прогноз —
через ``GET /forecast/{company_id}``, когда статус стал ``succeeded``.

Запуск из корня репозитория:

    uvicorn ml_service.app:app --host 0.0.0.0 --port 8000
"""

from __future__ import annotations

import logging
import math
import re
import threading
import uuid
from contextlib import asynccontextmanager
from datetime import datetime, timezone
from pathlib import Path

from fastapi import BackgroundTasks, FastAPI, File, Form, HTTPException, UploadFile

from . import config
from .preprocess import build_daily, read_statement, to_prophet_frame
from .registry import registry
from .schemas import (
    ForecastResponse,
    HealthResponse,
    JobStatus,
    ModelInfo,
    TrainResponse,
)
from .training import cash_gap, fit_final, forecast, tune

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
logger = logging.getLogger("ml_service")

# Ограничиваем число одновременных обучений: задача ждёт слот, оставаясь
# в статусе queued, чтобы бэкенд видел прогресс, а сервис не упирался в CPU.
_training_slots = threading.BoundedSemaphore(config.MAX_CONCURRENT_TRAININGS)

# company_id уходит в имя директории с моделью, поэтому пропускаем только
# буквы/цифры/«_»/«-»: точки и слэши закрывают выход за пределы artifacts/.
# \w в Python для str включает кириллицу, так что «С151» тоже валиден.
_COMPANY_ID_RE = re.compile(r"^[\w-]{1,64}$")
_COMPANY_ID_HINT = (
    "company_id: буквы (включая кириллицу), цифры, '_' и '-', от 1 до 64 "
    "символов; точки и слэши запрещены"
)


@asynccontextmanager
async def lifespan(_: FastAPI):
    # Фоновые задачи живут в памяти процесса: после рестарта их статус
    # остался бы running навсегда, и бэкенд ждал бы вечно.
    stale = registry.fail_unfinished_jobs("Сервис перезапущен во время обучения")
    if stale:
        logger.warning("после рестарта помечено failed: %d задач", stale)
    yield


app = FastAPI(
    title="Cashflow Forecast ML Service",
    summary="Прогноз остатка по расчётному счёту: Prophet + Optuna на каждую компанию.",
    version="0.1.0",
    lifespan=lifespan,
)


@app.get("/health", response_model=HealthResponse, tags=["service"])
def health() -> HealthResponse:
    return HealthResponse(
        status="ok",
        models=len(registry.list_models()),
        defaults={
            "n_trials": config.N_TRIALS,
            "top_k": config.TOP_K,
            "horizon_days": 30,
            "max_concurrent_trainings": config.MAX_CONCURRENT_TRAININGS,
        },
    )


@app.post("/train", response_model=TrainResponse, status_code=202, tags=["training"])
async def train(
    background_tasks: BackgroundTasks,
    company_id: str = Form(..., description="Идентификатор компании, например C022"),
    file: UploadFile = File(..., description="Выписка в формате кейса: CSV, разделитель ';'"),
    n_trials: int = Form(config.N_TRIALS, ge=1, le=100),
    top_k: int = Form(config.TOP_K, ge=1, le=10),
) -> TrainResponse:
    company_id = company_id.strip()
    if not _COMPANY_ID_RE.match(company_id):
        raise HTTPException(422, _COMPANY_ID_HINT)

    payload = await _read_upload(file)
    if not payload:
        raise HTTPException(422, "Загружен пустой файл")

    # Валидируем выписку до постановки задачи, чтобы бэкенд получил ошибку сразу,
    # а не через несколько минут в статусе job.
    try:
        daily = build_daily(read_statement(payload))
    except ValueError as exc:
        raise HTTPException(422, str(exc)) from exc
    except Exception as exc:  # noqa: BLE001
        raise HTTPException(422, f"Не удалось разобрать CSV: {exc}") from exc

    if len(daily) < config.MIN_HISTORY_DAYS:
        raise HTTPException(
            422,
            f"Слишком короткая история: {len(daily)} дней, минимум {config.MIN_HISTORY_DAYS}",
        )

    job_id = uuid.uuid4().hex
    upload_path = config.UPLOADS_DIR / f"{job_id}.csv"
    upload_path.write_bytes(payload)

    registry.create_job(job_id, company_id)
    background_tasks.add_task(_run_training, job_id, company_id, upload_path, n_trials, top_k)
    return TrainResponse(job_id=job_id, company_id=company_id, status="queued")


def _run_training(
    job_id: str, company_id: str, upload_path: Path, n_trials: int, top_k: int
) -> None:
    try:
        with _training_slots:
            registry.update_job(job_id, status="running", started_at=_stamp())
            _train(job_id, company_id, upload_path, n_trials, top_k)
    except Exception as exc:  # noqa: BLE001 — ошибка должна осесть в статусе задачи, а не уронить процесс
        logger.exception("job %s: обучение упало", job_id)
        registry.update_job(job_id, status="failed", finished_at=_stamp(), error=f"{type(exc).__name__}: {exc}")
    finally:
        upload_path.unlink(missing_ok=True)


def _train(job_id: str, company_id: str, upload_path: Path, n_trials: int, top_k: int) -> None:
    daily = build_daily(read_statement(upload_path))
    history = to_prophet_frame(daily)

    logger.info("job %s: тюнинг %s, %d дней, %d trials", job_id, company_id, len(history), n_trials)
    result = tune(history, n_trials=n_trials, top_k=top_k)

    model = fit_final(history, result["params"])
    meta = {
        "trained_at": _stamp(),
        "last_history_date": history["ds"].max().date().isoformat(),
        "last_balance": round(float(history["y"].iloc[-1]), 2),
        "n_days": int(len(history)),
        "params": result["params"],
        "metrics": _clean(result["metrics"]),
    }
    registry.save_model(company_id, model, meta)
    registry.update_job(
        job_id,
        status="succeeded",
        finished_at=_stamp(),
        params=result["params"],
        metrics=meta["metrics"],
    )
    logger.info("job %s: готово, holdout MAE=%.0f", job_id, result["metrics"]["mae_holdout"])


@app.get("/jobs/{job_id}", response_model=JobStatus, tags=["training"])
def job_status(job_id: str) -> JobStatus:
    if not re.fullmatch(r"[A-Za-z0-9_-]{1,64}", job_id):
        raise HTTPException(404, f"Задача {job_id} не найдена")
    try:
        return JobStatus(**registry.get_job(job_id))
    except FileNotFoundError:
        raise HTTPException(404, f"Задача {job_id} не найдена") from None


@app.get("/forecast/{company_id}", response_model=ForecastResponse, tags=["forecast"])
def get_forecast(company_id: str, horizon_days: int = 30) -> ForecastResponse:
    if not 1 <= horizon_days <= 365:
        raise HTTPException(422, "horizon_days должен быть от 1 до 365")
    company_id = _company_id_or_404(company_id)
    try:
        model, meta = registry.load_model(company_id)
    except FileNotFoundError:
        raise HTTPException(404, f"Модель для {company_id} ещё не обучена") from None

    fc = forecast(model, horizon_days)
    days = [
        {
            "date": row.ds.date().isoformat(),
            "balance": round(float(row.yhat), 2),
            "balance_lower": round(float(row.yhat_lower), 2),
            "balance_upper": round(float(row.yhat_upper), 2),
        }
        for row in fc.itertuples(index=False)
    ]
    gap = cash_gap(fc, config.GAP_PROBABILITY_THRESHOLD)
    return ForecastResponse(
        company_id=company_id,
        horizon_days=horizon_days,
        trained_at=meta["trained_at"],
        last_history_date=meta["last_history_date"],
        last_balance=meta["last_balance"],
        forecast=days,
        cash_gap=gap,
        summary=_summary(meta["last_balance"], days, gap, horizon_days),
        quality=meta["metrics"],
    )


@app.get("/models", response_model=list[ModelInfo], tags=["models"])
def list_models() -> list[dict]:
    return registry.list_models()


@app.get("/models/{company_id}", response_model=ModelInfo, tags=["models"])
def model_info(company_id: str) -> dict:
    company_id = _company_id_or_404(company_id)
    try:
        _, meta = registry.load_model(company_id)
    except FileNotFoundError:
        raise HTTPException(404, f"Модель для {company_id} не найдена") from None
    return {"company_id": company_id, **meta}


@app.delete("/models/{company_id}", status_code=204, tags=["models"])
def delete_model(company_id: str) -> None:
    company_id = _company_id_or_404(company_id)
    if not registry.delete_model(company_id):
        raise HTTPException(404, f"Модель для {company_id} не найдена")


def _stamp() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


async def _read_upload(file: UploadFile) -> bytes:
    """Читает файл кусками, чтобы не принять гигантский аплоад в память."""
    limit = config.MAX_UPLOAD_BYTES
    chunks: list[bytes] = []
    total = 0
    while chunk := await file.read(1024 * 1024):
        total += len(chunk)
        if total > limit:
            raise HTTPException(413, f"Файл больше {limit // (1024 * 1024)} МБ")
        chunks.append(chunk)
    return b"".join(chunks)


def _company_id_or_404(company_id: str) -> str:
    """Невалидный id не может соответствовать модели — отвечаем 404."""
    if not _COMPANY_ID_RE.match(company_id):
        raise HTTPException(404, f"Модель для {company_id} не найдена")
    return company_id


def _money(value: float) -> str:
    """1000000 -> '1 000 000' — так число читается в интерфейсе."""
    return f"{value:,.0f}".replace(",", " ")


def _summary(
    last_balance: float, days: list[dict], gap: dict | None, horizon: int
) -> str:
    """Короткий вывод простым языком: его показывает предпринимателю фронт."""
    start = days[0]["balance"]
    final = days[-1]["balance"]
    min_balance = min(day["balance"] for day in days)
    change = final - start
    direction = "вырастет" if change > 0 else "снизится"

    if gap:
        day_number = next(
            i for i, day in enumerate(days, 1) if day["date"] == gap["date"]
        )
        head = (
            f"Через {day_number} дн. (на {gap['date']}) денег может не хватить: "
            f"нехватка около {_money(gap['shortage'])} ₽, вероятность разрыва "
            f"{gap['probability']:.0%}. Минимальный остаток на горизонте — "
            f"{_money(min_balance)} ₽."
        )
    else:
        head = (
            f"На {horizon} дн. вперёд кассового разрыва не ожидается: "
            f"минимальный прогнозный остаток {_money(min_balance)} ₽."
        )

    if start:
        if change == 0:
            tail = f" За горизонт остаток останется на уровне {_money(final)} ₽."
        else:
            tail = (
                f" За горизонт остаток {direction} с {_money(start)} "
                f"до {_money(final)} ₽ ({change / abs(start):+.0%})."
            )
    else:
        tail = f" К концу горизонта ожидается {_money(final)} ₽."

    # Prophet строит тренд по всей истории, поэтому разовый шок последнего
    # дня он сглаживает. Об этом честнее сказать, чем показывать скачок молча.
    jump = abs(start - last_balance) / max(abs(last_balance), 1.0)
    if jump > 0.3:
        tail += (
            f" Прогноз начинается с {_money(start)} ₽, хотя последний факт — "
            f"{_money(last_balance)} ₽: модель сглаживает разовое движение "
            f"последнего дня."
        )
    return head + tail


def _clean(metrics: dict) -> dict:
    """NaN/inf в JSON не сериализуются — заменяем на None."""
    return {
        key: (None if isinstance(value, float) and not math.isfinite(value) else value)
        for key, value in metrics.items()
    }
