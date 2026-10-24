"""Смоук-тест живого сервиса: обучение -> статус -> прогноз.

Запуск при поднятом uvicorn из корня репозитория:

    .\\.venv\\Scripts\\python.exe -m ml_service.smoke_test
    .\\.venv\\Scripts\\python.exe -m ml_service.smoke_test --n-trials 1 --top-k 1

Скрипт намеренно ходит по HTTP, а не вызывает функции напрямую: так
проверяются multipart-загрузка, фоновые задачи и JSON-схемы ответов.
"""

from __future__ import annotations

import argparse
import math
import sys
import time
from pathlib import Path

import requests

BASE_URL = "http://127.0.0.1:8000"


def _fail(message: str) -> None:
    print(f"[FAIL] {message}")
    raise SystemExit(1)


def check_health(session: requests.Session) -> None:
    response = session.get(f"{BASE_URL}/health", timeout=10)
    if response.status_code != 200:
        _fail(f"/health вернул {response.status_code}: {response.text}")
    body = response.json()
    if body.get("status") != "ok":
        _fail(f"/health: неожиданный статус {body}")
    print(f"[ok] /health, обученных моделей: {body['models']}, дефолты: {body['defaults']}")


def train(
    session: requests.Session, csv_path: Path, company_id: str, n_trials: int, top_k: int
) -> str:
    with csv_path.open("rb") as payload:
        response = session.post(
            f"{BASE_URL}/train",
            data={"company_id": company_id, "n_trials": n_trials, "top_k": top_k},
            files={"file": (csv_path.name, payload, "text/csv")},
            timeout=60,
        )
    if response.status_code != 202:
        _fail(f"/train вернул {response.status_code}: {response.text}")
    body = response.json()
    job_id = body["job_id"]
    print(f"[ok] /train принял задачу {job_id} для {body['company_id']}")
    return job_id


def wait_job(session: requests.Session, job_id: str, timeout_s: int) -> dict:
    deadline = time.monotonic() + timeout_s
    last_status = None
    while time.monotonic() < deadline:
        response = session.get(f"{BASE_URL}/jobs/{job_id}", timeout=10)
        if response.status_code != 200:
            _fail(f"/jobs/{job_id} вернул {response.status_code}: {response.text}")
        job = response.json()
        if job["status"] != last_status:
            print(f"[..] задача {job_id}: {job['status']}")
            last_status = job["status"]
        if job["status"] == "succeeded":
            return job
        if job["status"] == "failed":
            _fail(f"обучение упало: {job.get('error')}")
        time.sleep(2)
    _fail(f"задача {job_id} не завершилась за {timeout_s} с (статус {last_status})")


def check_forecast(session: requests.Session, company_id: str, horizon: int) -> None:
    response = session.get(
        f"{BASE_URL}/forecast/{company_id}", params={"horizon_days": horizon}, timeout=60
    )
    if response.status_code != 200:
        _fail(f"/forecast/{company_id} вернул {response.status_code}: {response.text}")
    body = response.json()

    days = body["forecast"]
    if len(days) != horizon:
        _fail(f"в прогнозе {len(days)} дней, ожидали {horizon}")

    expected_start = _next_day(body["last_history_date"])
    if days[0]["date"] != expected_start:
        _fail(f"прогноз начинается {days[0]['date']}, ожидали {expected_start}")

    for day in days:
        for key in ("balance", "balance_lower", "balance_upper"):
            if not math.isfinite(day[key]):
                _fail(f"{day['date']}: {key} не число ({day[key]})")
        if not day["balance_lower"] <= day["balance"] <= day["balance_upper"]:
            _fail(f"{day['date']}: интервал не накрывает прогноз: {day}")

    first, last = days[0], days[-1]
    print(
        f"[ok] /forecast/{company_id}: {len(days)} дней, "
        f"{first['date']}..{last['date']}, "
        f"баланс {first['balance']:,.0f} -> {last['balance']:,.0f} руб."
    )
    print(f"[ok] качество: {body['quality']}")
    print(f"[ok] вывод: {body['summary']}")
    gap = body["cash_gap"]
    if gap is None:
        print("[ok] кассового разрыва на горизонте не ожидается")
    else:
        print(
            f"[ok] кассовый разрыв {gap['date']}: нехватка {gap['shortage']:,.0f} руб., "
            f"вероятность {gap['probability']:.0%}"
        )

    models = session.get(f"{BASE_URL}/models", timeout=10).json()
    if not any(m["company_id"] == company_id for m in models):
        _fail(f"{company_id} нет в /models после успешного обучения")
    print(f"[ok] /models: {len(models)} моделей, {company_id} на месте")


def check_errors(session: requests.Session) -> None:
    """Негативные сценарии: бэкенд должен получать 422/404, а не 500."""
    broken = "company_id;operation_id\nC999;1\n"
    response = session.post(
        f"{BASE_URL}/train",
        data={"company_id": "C999", "n_trials": 1, "top_k": 1},
        files={"file": ("broken.csv", broken.encode("utf-8"), "text/csv")},
        timeout=30,
    )
    if response.status_code != 422:
        _fail(f"битый CSV: ждали 422, получили {response.status_code}")
    print(f"[ok] битый CSV -> 422: {response.json()['detail']}")

    short = "company_id;operation_id;operation_date;amount;direction;counterparty_inn;counterparty_name;payment_purpose;balance_eod\n"
    short += "".join(
        f"C999;{i};2026-01-{i:02d};1000;Поступление;7700000000;ООО Тест;тест;{1000 * i}\n"
        for i in range(1, 11)
    )
    response = session.post(
        f"{BASE_URL}/train",
        data={"company_id": "C999", "n_trials": 1, "top_k": 1},
        files={"file": ("short.csv", short.encode("utf-8"), "text/csv")},
        timeout=30,
    )
    if response.status_code != 422:
        _fail(f"короткая история: ждали 422, получили {response.status_code}")
    print(f"[ok] короткая история -> 422: {response.json()['detail']}")

    for path in ("/forecast/NO_SUCH_COMPANY", "/models/NO_SUCH_COMPANY", "/jobs/deadbeef"):
        response = session.get(f"{BASE_URL}{path}", timeout=10)
        if response.status_code != 404:
            _fail(f"{path}: ждали 404, получили {response.status_code}")
    print("[ok] неизвестная компания/задача -> 404")


def _next_day(iso_date: str) -> str:
    from datetime import date, timedelta

    return (date.fromisoformat(iso_date) + timedelta(days=1)).isoformat()


def main() -> int:
    global BASE_URL
    # Консоль Windows по умолчанию cp1251 — кириллица в отчёте превращается в мусор.
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--csv", type=Path, default=Path("data/statements/C022.csv"))
    parser.add_argument("--company", default="C022")
    parser.add_argument("--horizon", type=int, default=30)
    parser.add_argument("--n-trials", type=int, default=1, help="для смоука хватает 1")
    parser.add_argument("--top-k", type=int, default=1)
    parser.add_argument("--timeout", type=int, default=900)
    parser.add_argument("--base-url", default=BASE_URL)
    args = parser.parse_args()

    BASE_URL = args.base_url.rstrip("/")

    if not args.csv.exists():
        _fail(f"нет файла {args.csv}")

    with requests.Session() as session:
        check_health(session)
        check_errors(session)
        job_id = train(session, args.csv, args.company, args.n_trials, args.top_k)
        job = wait_job(session, job_id, args.timeout)
        print(f"[ok] задача {job_id}: params={job['params']}")
        check_forecast(session, args.company, args.horizon)

    print("\nСмоук-тест пройден.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
