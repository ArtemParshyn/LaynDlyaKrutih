from typing import Literal

from pydantic import BaseModel, Field

from . import config


class TrainResponse(BaseModel):
    job_id: str
    company_id: str
    status: Literal["queued"]


class JobStatus(BaseModel):
    job_id: str
    company_id: str
    status: Literal["queued", "running", "succeeded", "failed"]
    created_at: str
    started_at: str | None = None
    finished_at: str | None = None
    error: str | None = None
    metrics: dict | None = None
    params: dict | None = None


class DailyBalance(BaseModel):
    date: str
    balance: float = Field(..., description="Прогноз остатка на конец дня, ₽")
    balance_lower: float = Field(..., description="Нижняя граница 80% интервала")
    balance_upper: float = Field(..., description="Верхняя граница 80% интервала")


class CashGap(BaseModel):
    date: str = Field(..., description="Первый день, когда денег может не хватить")
    shortage: float = Field(..., description="На сколько ₽ прогноз уходит ниже нуля")
    probability: float = Field(..., ge=0, le=1, description="Оценка вероятности разрыва")
    balance_forecast: float


class ForecastResponse(BaseModel):
    company_id: str
    horizon_days: int
    trained_at: str
    last_history_date: str
    last_balance: float = Field(..., description="Последний известный остаток, ₽")
    forecast: list[DailyBalance]
    cash_gap: CashGap | None = Field(
        None, description="null, если разрыва на горизонте не ожидается"
    )
    summary: str = Field(..., description="Вывод простым языком — его можно показать предпринимателю")
    quality: dict = Field(..., description="Метрики на holdout при обучении")
    # Новая ML-версия возвращает структурированные риски и объяснения.
    # Оставляем тип гибким для совместимости с list/dict форматами.
    risks: list | dict | None = None
    explanations: list | dict | None = None


class ModelInfo(BaseModel):
    company_id: str
    trained_at: str
    last_history_date: str
    last_balance: float
    n_days: int
    params: dict
    metrics: dict


class HealthResponse(BaseModel):
    status: Literal["ok"]
    models: int = Field(..., description="Сколько компаний уже обучено")
    defaults: dict
