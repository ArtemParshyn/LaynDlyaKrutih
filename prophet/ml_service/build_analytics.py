"""Собрать разметку по всем выпискам из data/statements.

Нужно, чтобы ``GET /analytics/{company_id}`` работал для компаний, которые
обучали до появления категоризации: модели есть, а разметки ещё нет.

    .\\.venv\\Scripts\\python.exe -m ml_service.build_analytics
"""

from __future__ import annotations

import argparse
from pathlib import Path

from . import analytics, config, preprocess


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--data-dir", type=Path, default=config.BASE_DIR / "data" / "statements"
    )
    args = parser.parse_args()

    files = sorted(path for path in args.data_dir.glob("*.csv") if path.is_file())
    if not files:
        print(f"нет CSV в {args.data_dir}")
        return 1

    total_ops = 0
    other_ops = 0
    for index, path in enumerate(files, 1):
        company_id = path.stem
        operations = preprocess.read_statement(path)
        categorized = analytics.categorize_operations(operations, company_id)
        summary = analytics.summarize(categorized, company_id)
        analytics.save(company_id, categorized, summary)
        total_ops += len(categorized)
        other_ops += int((categorized["category"] == "Прочее").sum())
        print(
            f"[{index}/{len(files)}] {company_id}: {len(categorized)} операций, "
            f"«Прочее» {summary['other_share_amount']:.1%} денег",
            flush=True,
        )

    print(
        f"\nготово: {len(files)} компаний, {total_ops} операций, "
        f"«Прочее» {other_ops / max(total_ops, 1):.2%} операций"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
