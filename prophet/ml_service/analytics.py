"""Категоризация операций и хранение аналитики по компании.

Разметка считается при обучении (``POST /train``) и складывается рядом
с моделью, чтобы фронт мог показать структуру расходов без повторной
загрузки выписки.
"""

from __future__ import annotations

import json
from functools import lru_cache
from pathlib import Path

import pandas as pd

from . import categorize, config, recurring

ANALYTICS_DIR = config.ARTIFACTS_DIR / "analytics"


@lru_cache(maxsize=1)
def _companies() -> pd.DataFrame:
    path = config.BASE_DIR / "data" / "companies.csv"
    if not path.exists():
        return pd.DataFrame(columns=["company_id", "company_inn", "industry"])
    return pd.read_csv(
        path,
        sep=";",
        encoding="utf-8-sig",
        dtype={"company_id": str, "company_inn": str},
    )


def categorize_operations(operations: pd.DataFrame, company_id: str) -> pd.DataFrame:
    """Размечает операции компании по категориям."""
    return categorize.classify(operations, _companies())


def summarize(
    categorized: pd.DataFrame, company_id: str, sample_size: int = 20
) -> dict:
    total_ops = int(len(categorized))
    total_amount = float(categorized["amount"].sum())

    categories = []
    grouped = (
        categorized.groupby("category")
        .agg(ops=("amount", "size"), amount=("amount", "sum"))
        .sort_values("amount", ascending=False)
    )
    for name, row in grouped.iterrows():
        categories.append(
            {
                "category": name,
                "ops": int(row.ops),
                "amount": round(float(row.amount), 2),
                "share_ops": round(float(row.ops) / total_ops, 4) if total_ops else 0.0,
                "share_amount": round(float(row.amount) / total_amount, 4)
                if total_amount
                else 0.0,
            }
        )

    sources = {
        str(name): int(count)
        for name, count in categorized["source"].value_counts().items()
    }

    other = grouped.loc["Прочее"] if "Прочее" in grouped.index else None
    other_share_ops = round(float(other.ops) / total_ops, 4) if other is not None and total_ops else 0.0
    other_share_amount = (
        round(float(other.amount) / total_amount, 4)
        if other is not None and total_amount
        else 0.0
    )

    revenue = categorized[categorized["category"] == "Выручка"]
    revenue_subtypes = {
        str(name): {
            "ops": int(count),
            "amount": round(float(revenue.loc[revenue.subcategory == name, "amount"].sum()), 2),
        }
        for name, count in revenue["subcategory"].value_counts().items()
    }

    top_counterparties: dict[str, list[dict]] = {}
    for category in grouped.index[:8]:
        subset = categorized[categorized["category"] == category]
        top = subset.groupby("counterparty_name")["amount"].sum().nlargest(3)
        top_counterparties[str(category)] = [
            {"name": str(name), "amount": round(float(amount), 2)}
            for name, amount in top.items()
        ]

    sample_rows = (
        categorized.sort_values("operation_date")
        .groupby("category", group_keys=False)
        .head(2)
        .head(sample_size)
    )
    sample = [
        {
            "date": pd.Timestamp(row.operation_date).date().isoformat(),
            "amount": round(float(row.amount), 2),
            "direction": str(row.direction),
            "counterparty": str(row.counterparty_name),
            "role": str(row.role),
            "category": str(row.category),
            "subcategory": str(row.subcategory),
            "source": str(row.source),
            "confidence": round(float(row.confidence), 2),
        }
        for row in sample_rows.itertuples(index=False)
    ]

    return {
        "company_id": company_id,
        "operations": total_ops,
        "amount": round(total_amount, 2),
        "last_history_date": (
            pd.to_datetime(categorized["operation_date"]).max().date().isoformat()
            if total_ops
            else None
        ),
        "categories": categories,
        "sources": sources,
        "other_share_ops": other_share_ops,
        "other_share_amount": other_share_amount,
        "revenue_subtypes": revenue_subtypes,
        "top_counterparties": top_counterparties,
        "sample": sample,
    }


def save(company_id: str, categorized: pd.DataFrame, summary: dict) -> Path:
    """Сохраняет разметку и сводку в artifacts/analytics/{company_id}/."""
    directory = ANALYTICS_DIR / company_id
    directory.mkdir(parents=True, exist_ok=True)

    columns = [
        "company_id", "operation_id", "operation_date", "amount", "direction",
        "counterparty_inn", "counterparty_name", "role", "industry",
        "category", "subcategory", "source", "confidence",
    ]
    categorized[columns].to_csv(
        directory / "operations.csv", index=False, sep=";", encoding="utf-8-sig"
    )
    (directory / "summary.json").write_text(
        json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8"
    )
    (directory / "recurring.json").write_text(
        json.dumps(recurring.detect_streams(categorized), ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    return directory


def load_summary(company_id: str) -> dict:
    path = ANALYTICS_DIR / company_id / "summary.json"
    if not path.exists():
        raise FileNotFoundError(company_id)
    return json.loads(path.read_text(encoding="utf-8"))


def load_operations(company_id: str) -> pd.DataFrame:
    path = ANALYTICS_DIR / company_id / "operations.csv"
    if not path.exists():
        raise FileNotFoundError(company_id)
    return pd.read_csv(
        path, sep=";", encoding="utf-8-sig", dtype={"company_id": str, "counterparty_inn": str}
    )


def load_recurring(company_id: str) -> list[dict]:
    path = ANALYTICS_DIR / company_id / "recurring.json"
    if not path.exists():
        raise FileNotFoundError(company_id)
    return json.loads(path.read_text(encoding="utf-8"))


def last_history_date(company_id: str) -> str | None:
    summary = load_summary(company_id)
    if summary.get("last_history_date"):
        return summary["last_history_date"]
    operations = load_operations(company_id)
    if operations.empty:
        return None
    return pd.to_datetime(operations["operation_date"]).max().date().isoformat()


def list_companies() -> list[str]:
    if not ANALYTICS_DIR.exists():
        return []
    return sorted(
        path.name for path in ANALYTICS_DIR.iterdir() if (path / "summary.json").exists()
    )
