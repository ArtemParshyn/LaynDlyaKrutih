from datetime import date

from pydantic import BaseModel, Field


class HistoryDay(BaseModel):
    date: date
    irregular_in: int = Field(ge=0)
    irregular_out: int = Field(ge=0)


class ScheduledPayment(BaseModel):
    date: date
    amount: int = Field(gt=0)
    is_income: bool
    category: str = Field(min_length=1)


class ForecastRequest(BaseModel):
    as_of: date
    opening_balance: int
    history: list[HistoryDay]
    scheduled: list[ScheduledPayment]


class ForecastDay(BaseModel):
    date: date
    expected_balance: int
    balance_low: int
    balance_high: int
    inflow: int
    outflow: int


class Alert(BaseModel):
    date: date
    shortfall: int = Field(ge=0)
    chance: float = Field(ge=0, le=1)
    reason: str = Field(min_length=1)


class RarePayment(BaseModel):
    """A large transfer that repeats, but not often enough to be a daily flow."""

    amount: int = Field(gt=0)
    is_income: bool
    seen: int = Field(ge=2)
    roughly_every_days: int = Field(ge=1)


class ForecastResponse(BaseModel):
    days: list[ForecastDay]
    alerts: list[Alert]
    rough: bool = False
    rare_payments: list[RarePayment] = Field(default_factory=list)
