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
    risk: bool = Field(..., description="Хватает ли денег на обязательные платежи")
    reason: str | None = Field(
        None, description="negative_balance — остаток уже ниже нуля; payment_shortfall — не хватает на платёж"
    )
    date: str | None = Field(None, description="Первый день, когда денег может не хватить")
    shortage: float | None = Field(None, description="Сколько ₽ не хватает в худший день")
    shortage_stressed: float | None = Field(
        None, description="Дефицит по нижней границе 80% интервала"
    )
    probability: float | None = Field(
        None, ge=0, le=1, description="Вероятность, что остатка не хватит на платёж"
    )
    required_payment: float | None = Field(None, description="Платёж, на который не хватает, ₽")
    balance_forecast: float | None = Field(None, description="Прогноз остатка в этот день, ₽")
    worst_date: str | None = Field(None, description="День максимального дефицита")
    max_probability: float = Field(0.0, ge=0, le=1, description="Максимальная вероятность за горизонт")
    negative_balance_days: int = Field(0, description="Сколько дней прогноз остатка ниже нуля")
    payment_shortfall_days: int = Field(0, description="Сколько дней не хватает на платёж")
    horizon_days: int = Field(30, description="Горизонт оценки")


class ForecastResponse(BaseModel):
    company_id: str
    horizon_days: int
    trained_at: str
    last_history_date: str
    last_balance: float = Field(..., description="Последний известный остаток, ₽")
    forecast: list[DailyBalance]
    cash_gap: CashGap | None = Field(
        None, description="Оценка риска: дата, сумма, вероятность; null только если аналитики нет"
    )
    summary: str = Field(..., description="Вывод простым языком — его можно показать предпринимателю")
    quality: dict = Field(..., description="Метрики на holdout при обучении")


class ModelInfo(BaseModel):
    company_id: str
    trained_at: str
    last_history_date: str
    last_balance: float
    n_days: int
    params: dict
    metrics: dict


class CategoryLine(BaseModel):
    category: str
    ops: int
    amount: float
    share_ops: float
    share_amount: float


class CategorySample(BaseModel):
    date: str
    amount: float
    direction: str
    counterparty: str
    role: str
    category: str
    subcategory: str
    source: str
    confidence: float


class CategoryReport(BaseModel):
    company_id: str
    operations: int
    amount: float
    last_history_date: str | None = None
    categories: list[CategoryLine]
    sources: dict[str, int]
    other_share_ops: float
    other_share_amount: float
    revenue_subtypes: dict[str, dict]
    top_counterparties: dict[str, list[dict]]
    sample: list[CategorySample] = []


class RecurringStream(BaseModel):
    stream_id: str
    company_id: str
    direction: str
    category: str
    subcategory: str
    counterparty: str
    counterparty_name: str
    period_days: int
    period: str
    anchor_weekday: int | None = None
    anchors: list[int] = []
    occurrences: int
    first_date: str
    last_date: str
    next_date: str
    window_days: int
    amount_median: float
    amount_low: float
    amount_high: float
    amount_cv: float
    interval_cv: float
    coverage: float
    status: str
    mandatory: bool
    confidence: float


class CalendarPayment(BaseModel):
    date: str
    category: str
    subcategory: str
    counterparty: str
    amount: float
    amount_low: float
    amount_high: float
    mandatory: bool
    confidence: float
    overdue: bool = False


class CalendarResponse(BaseModel):
    company_id: str
    start: str
    end: str
    days: int
    total: float
    payments: list[CalendarPayment]


class HealthResponse(BaseModel):
    status: Literal["ok"]
    models: int = Field(..., description="Сколько компаний уже обучено")
    defaults: dict
