"""Сравнение качества модели при разном ``n_trials``.

Данные и ``top_k`` фиксированы — меняется только число trials Optuna.
Так видно, окупается ли увеличение перебора.

Запуск при поднятом сервисе, из корня репозитория:

    .\\.venv\\Scripts\\python.exe -m ml_service.tune_sweep
    .\\.venv\\Scripts\\python.exe -m ml_service.tune_sweep --trials 1 5 20 --top-k 2

Результаты пишутся в CSV после каждого прогона: если прервать, уже
собранные строки останутся.
"""

from __future__ import annotations

import argparse
import csv
import time
from pathlib import Path

import requests

COLUMNS = [
    "n_trials",
    "status",
    "elapsed_s",
    "mae_screen",
    "mae_cv",
    "mape_cv",
    "mae_holdout",
    "mape_holdout",
    "smape_holdout",
    "params",
    "error",
]


def _error_text(response: requests.Response) -> str:
    try:
        detail = response.json().get("detail")
    except ValueError:
        return response.text
    if isinstance(detail, str):
        return detail
    if isinstance(detail, list):
        return "; ".join(
            f"{'.'.join(map(str, item.get('loc', [])))}: {item.get('msg')}"
            for item in detail
        )
    return str(detail)


def _run_one(
    base_url: str, company_id: str, csv_path: Path, n_trials: int, top_k: int
) -> dict:
    started = time.monotonic()
    with csv_path.open("rb") as payload:
        response = requests.post(
            f"{base_url}/train",
            data={"company_id": company_id, "n_trials": n_trials, "top_k": top_k},
            files={"file": (csv_path.name, payload, "text/csv")},
            timeout=120,
        )
    if response.status_code != 202:
        return {
            "n_trials": n_trials,
            "status": f"http_{response.status_code}",
            "error": _error_text(response),
            "elapsed_s": round(time.monotonic() - started, 1),
        }

    job_id = response.json()["job_id"]
    deadline = time.monotonic() + 1800
    while time.monotonic() < deadline:
        job = requests.get(f"{base_url}/jobs/{job_id}", timeout=10).json()
        if job["status"] in ("succeeded", "failed"):
            break
        time.sleep(2)
    else:
        return {
            "n_trials": n_trials,
            "status": "timeout",
            "error": f"задача {job_id} не завершилась",
            "elapsed_s": round(time.monotonic() - started, 1),
        }

    elapsed = round(time.monotonic() - started, 1)
    if job["status"] == "failed":
        return {"n_trials": n_trials, "status": "failed", "error": job["error"], "elapsed_s": elapsed}

    metrics = job.get("metrics") or {}
    params = job.get("params") or {}
    return {
        "n_trials": n_trials,
        "status": "succeeded",
        "elapsed_s": elapsed,
        "mae_screen": metrics.get("mae_screen"),
        "mae_cv": metrics.get("mae_cv"),
        "mape_cv": metrics.get("mape_cv"),
        "mae_holdout": metrics.get("mae_holdout"),
        "mape_holdout": metrics.get("mape_holdout"),
        "smape_holdout": metrics.get("smape_holdout"),
        "params": (
            f"cps={params.get('cps'):.5f} sps={params.get('sps'):.3f} "
            f"mode={params.get('mode')} weekly={params.get('weekly')}"
            if params
            else None
        ),
        "error": None,
    }


def _fmt(value) -> str:
    if value is None:
        return "—"
    if isinstance(value, float):
        return f"{value:,.0f}".replace(",", " ") if abs(value) >= 1000 else f"{value:.2f}"
    return str(value)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--csv", type=Path, default=Path("data/statements/C022.csv"))
    parser.add_argument("--company", default="C022", help="модель перезаписывается на каждом шаге")
    parser.add_argument("--top-k", type=int, default=2)
    parser.add_argument("--trials", type=int, nargs="+", default=[1, 2, 3, 5, 10, 20, 50])
    parser.add_argument("--base-url", default="http://127.0.0.1:8000")
    parser.add_argument(
        "--out", type=Path, default=Path("artifacts/experiments/n_trials_comparison.csv")
    )
    args = parser.parse_args()

    if not args.csv.exists():
        print(f"нет файла {args.csv}")
        return 1

    args.out.parent.mkdir(parents=True, exist_ok=True)
    rows: list[dict] = []
    with args.out.open("w", newline="", encoding="utf-8") as csv_file:
        writer = csv.DictWriter(csv_file, fieldnames=COLUMNS)
        writer.writeheader()

        for n_trials in args.trials:
            print(f"\n=== n_trials={n_trials}, top_k={args.top_k} ===", flush=True)
            row = _run_one(args.base_url, args.company, args.csv, n_trials, args.top_k)
            print(
                f"{row['status']}: {row.get('elapsed_s')} с, "
                f"mae_cv={_fmt(row.get('mae_cv'))}, "
                f"mae_holdout={_fmt(row.get('mae_holdout'))}, "
                f"{row.get('params') or row.get('error') or ''}",
                flush=True,
            )
            writer.writerow({key: row.get(key) for key in COLUMNS})
            csv_file.flush()
            rows.append(row)

    header = f"{'n_trials':>8} {'время, с':>9} {'MAE CV':>12} {'MAE holdout':>13} {'sMAPE, %':>10}"
    print("\n" + header)
    print("-" * len(header))
    for row in rows:
        print(
            f"{row['n_trials']:>8} {row.get('elapsed_s', ''):>9} "
            f"{_fmt(row.get('mae_cv')):>12} {_fmt(row.get('mae_holdout')):>13} "
            f"{_fmt(row.get('smape_holdout')):>10}"
        )
    print(f"\nCSV: {args.out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
