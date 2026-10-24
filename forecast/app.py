from fastapi import FastAPI, HTTPException

from build import build
from check import NotEnoughData, check
from contract import ForecastRequest, ForecastResponse

app = FastAPI()


@app.get("/health")
def health() -> dict:
    return {"status": "ok"}


@app.post("/forecast", response_model=ForecastResponse)
def forecast(request: ForecastRequest) -> ForecastResponse:
    try:
        enough = check(request)
    except NotEnoughData as error:
        raise HTTPException(status_code=422, detail=str(error)) from error

    days, alerts, rare = build(request, rough=not enough)
    return ForecastResponse(days=days, alerts=alerts, rough=not enough, rare_payments=rare)
