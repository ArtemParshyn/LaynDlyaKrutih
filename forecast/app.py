from datetime import timedelta

from fastapi import FastAPI, HTTPException

from check import NotEnoughData, check
from contract import ForecastDay, ForecastRequest, ForecastResponse

app = FastAPI()
HORIZON_DAYS = 30


@app.get("/health")
def health() -> dict:
    return {"status": "ok"}


@app.post("/forecast")
def forecast(request: ForecastRequest) -> ForecastResponse:
    try:
        rough = not check(request)
    except NotEnoughData as error:
        raise HTTPException(status_code=400, detail=str(error)) from error

    days = [
        ForecastDay(
            date=request.as_of + timedelta(days=offset),
            expected_balance=0,
            balance_low=0,
            balance_high=0,
            inflow=0,
            outflow=0,
        )
        for offset in range(1, HORIZON_DAYS + 1)
    ]
    return ForecastResponse(days=days, alerts=[], rough=rough)
