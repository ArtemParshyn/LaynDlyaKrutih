"""Детерминированная категоризация операций без ML.

Порядок принятия решения:

1. Контрагент — сама компания (по ИНН) -> внутренние переводы.
2. Особые контрагенты: банк, ФНС, учредитель, сотрудники, физлица.
3. Отрасль контрагента + направление операции -> категория.
4. Сглаживание: доминирующая категория контрагента исправляет выбросы.

Текст назначения используется только как уточнение подкатегории и в тех
случаях, где отрасль бессильна (банк).

Запуск:

    .\\.venv\\Scripts\\python.exe -m ml_service.categorize
    .\\.venv\\Scripts\\python.exe -m ml_service.categorize --company C022
"""

from __future__ import annotations

import argparse
import re
from pathlib import Path

import pandas as pd

from . import taxonomy

LEGAL_FORM = re.compile(r"^(ООО|АО|ПАО|ЗАО|ИП)\s*")
TRAILING_NUMBER = re.compile(r"[-–\s]*\d+$")

CONSTRUCTION_INDUSTRY = "Строительство и ремонт"
AUTO_INDUSTRY = "Автосервис"
CLINIC_INDUSTRY = "Медицинская клиника / стоматология"

CONSTRUCTION_ROLES = {
    "Стройподряд", "Генподрядчик", "Ремонт-Строй", "Электромонтаж",
    "Сантехмонтаж", "Фасад-Сервис", "Отделка-Профи",
}

# Сглаживание не применяем к категориям, где законна внутренняя вариативность
NO_SMOOTH_CATEGORIES = {"Выручка", "Возвраты", "Внутренние и владелец"}
SMOOTH_MIN_OPS = 5
SMOOTH_MIN_SHARE = 0.85


def normalize_role(name: str) -> str:
    """'ООО «Арендодатель-022»' -> 'Арендодатель'."""
    if pd.isna(name):
        return ""
    text = LEGAL_FORM.sub("", str(name)).strip()
    text = text.strip(" «»\"")
    text = TRAILING_NUMBER.sub("", text).strip(" «»\"")
    return text


def _text(value) -> str:
    return str(value).lower().replace("ё", "е") if pd.notna(value) else ""


def _revenue_subtype(industry: str, purpose: str) -> str:
    if industry == "government_customer":
        return "госзакупки"
    if industry == "marketplace":
        return "маркетплейсы"
    if industry == "individual":
        return "B2C"
    if industry == "grant":
        return "гранты и субсидии"
    if "эквайринг" in purpose:
        return "эквайринг"
    if "сбп" in purpose:
        return "СБП"
    if "торговая выручка" in purpose:
        return "наличная выручка"
    return "счета B2B"


def _tax_subtype(purpose: str) -> str:
    if "енп" in purpose or "единый налоговый" in purpose:
        return "ЕНП"
    if "усн" in purpose:
        return "УСН"
    if "ндфл" in purpose:
        return "НДФЛ"
    if "взнос" in purpose or "страхов" in purpose:
        return "страховые взносы"
    return "прочие налоги"


def _loan_subtype(role: str, purpose: str) -> str:
    if role in taxonomy.LEASING:
        return "лизинг"
    if "процент" in purpose:
        return "проценты"
    return "тело кредита"


def _bank(direction: str, purpose: str) -> tuple[str, str, str, float]:
    if direction == "Поступление":
        if "эквайринг" in purpose:
            return "Выручка", "эквайринг", "special", 0.98
        if "сбп" in purpose:
            return "Выручка", "СБП", "special", 0.98
        if "кредит" in purpose or "займ" in purpose:
            return "Кредиты и лизинг", "получение кредита/займа", "special", 0.9
        return "Выручка", "прочие поступления", "special", 0.6
    if "комисси" in purpose:
        return "Административные", "банковские комиссии", "special", 0.95
    if "заработн" in purpose or "зарплат" in purpose or "реестр" in purpose:
        return "Зарплата", "сотрудники", "special", 0.95
    if "кредит" in purpose or "процент" in purpose or "погашени" in purpose:
        return "Кредиты и лизинг", _loan_subtype("", purpose), "special", 0.9
    if "pos" in purpose or "терминал" in purpose:
        return "Аренда", "POS-терминалы", "special", 0.9
    return "Административные", "банковские услуги", "special", 0.5


def classify_row(row) -> tuple[str, str, str, float]:
    role = row["role"]
    industry = row["industry"]
    direction = row["direction"]
    purpose = row["purpose"]
    company_industry = row["company_industry"]

    # 1. Контрагент — сама компания: перевод между своими счетами
    if row["is_self"]:
        return "Внутренние и владелец", "свои счета", "self_inn", 1.0

    # 2. Особые контрагенты
    if industry == "founder":
        if "займ" in purpose or "кредит" in purpose:
            return "Внутренние и владелец", "займы учредителя", "special", 0.95
        return "Внутренние и владелец", "владельцу", "special", 0.9

    if industry == "owner":
        if direction == "Поступление" and ("торговая выручка" in purpose or "выручк" in purpose):
            return "Выручка", "наличная выручка", "special", 0.9
        return "Внутренние и владелец", "владельцу", "special", 0.95

    if industry == "employee":
        return "Зарплата", "сотрудники", "special", 0.98

    if industry == "self_employed":
        return "Зарплата", "подрядчики (ГПХ)", "special", 0.95

    if industry == "individual":
        if direction == "Поступление":
            return "Выручка", "B2C", "industry", 0.9
        return "Прочее", "выплаты физлицам", "industry", 0.6

    if industry == "bank":
        return _bank(direction, purpose)

    if industry == "taxes":
        if direction == "Поступление":
            return "Возвраты", "налоги", "special", 0.8
        return "Налоги", _tax_subtype(purpose), "special", 0.98

    # Возвраты ловим до отраслевых правил: «возврат ошибочного платежа»
    # не должен становиться закупкой или выплатой клиенту.
    if "возврат" in purpose:
        subtype = "клиентам" if direction == "Списание" else "от поставщиков"
        return "Возвраты", subtype, "purpose", 0.8

    if industry == "grant":
        return "Выручка", "гранты и субсидии", "special", 0.95

    # 3. Поступления: почти всё, что пришло не от финансовых контрагентов, — выручка
    if direction == "Поступление":
        if "займ" in purpose or "кредит" in purpose:
            return "Кредиты и лизинг", "получение кредита/займа", "purpose", 0.8
        return "Выручка", _revenue_subtype(industry, purpose), "industry", 0.9

    # 4. Списания: категория по отрасли контрагента
    if industry == "rent":
        subtype = "POS-терминалы" if "pos" in purpose or "терминал" in purpose else "помещение"
        return "Аренда", subtype, "industry", 0.95

    if industry == "utilities":
        if role == "Энергосбыт":
            subtype = "электроэнергия"
        elif role == "Водоканал":
            subtype = "вода"
        else:
            subtype = "вывоз ТКО"
        return "Коммунальные", subtype, "industry", 0.95

    if industry == "telecom_it":
        subtype = "телеком" if role == "Телеком" else "SaaS и подписки"
        return "Связь и IT", subtype, "industry", 0.95

    if industry == "marketing":
        return "Маркетинг", "реклама и продвижение", "industry", 0.95

    if industry == "logistics":
        if role == "Топливная компания":
            subtype = "топливо"
        elif role == "Система взимания платы":
            subtype = "платные дороги"
        else:
            subtype = "доставка и экспедирование"
        return "Логистика", subtype, "industry", 0.9

    if industry == "capex":
        return "Оборудование", "техника и оборудование", "industry", 0.9

    if industry == "leasing":
        return "Кредиты и лизинг", "лизинг", "industry", 0.95

    if industry == "insurance":
        return "Административные", "страхование", "industry", 0.85

    if industry == "office":
        return "Административные", "офис и хозтовары", "industry", 0.85

    if industry == "med_lab":
        if company_industry == CLINIC_INDUSTRY:
            return "Себестоимость", "медицинские услуги", "industry", 0.85
        return "Административные", "лабораторные услуги", "industry", 0.75

    if industry == "auto_service":
        if company_industry == AUTO_INDUSTRY:
            return "Себестоимость", "запчасти и автоуслуги", "industry", 0.85
        return "Административные", "ремонт и обслуживание", "industry", 0.8

    if role in CONSTRUCTION_ROLES:
        if company_industry == CONSTRUCTION_INDUSTRY:
            return "Себестоимость", "подрядчики", "industry", 0.85
        return "Административные", "ремонт и обслуживание", "industry", 0.8

    if industry == "admin":
        if role == "Истец":
            subtype = "юристы и суды"
        elif role == "Бухгалтерия-Аутсорс":
            subtype = "бухгалтерия"
        elif role in {"Клининг", "Клининг-Сервис"}:
            subtype = "клининг"
        elif role == "Охранное предприятие":
            subtype = "охрана"
        else:
            subtype = "ремонт и обслуживание"
        return "Административные", subtype, "industry", 0.9

    if industry == "food_supplier":
        return "Себестоимость", "товары и сырьё", "industry", 0.9

    if industry == "other_supplier":
        subtype = "упаковка" if role == "Упаковка-Юг" else "материалы и товары"
        return "Себестоимость", subtype, "industry", 0.9

    if industry == "customer":
        if "возврат" in purpose:
            return "Возвраты", "клиентам", "purpose", 0.8
        return "Прочее", "выплаты клиентам", "industry", 0.5

    if industry == "government_customer":
        return "Прочее", "расчёты с бюджетом", "industry", 0.6

    return "Прочее", "не распознано", "fallback", 0.3


def classify(operations: pd.DataFrame, companies: pd.DataFrame) -> pd.DataFrame:
    """Возвращает копию операций с категориями, подкатегориями и источником."""
    ops = operations.copy()

    company_info = companies.set_index("company_id")
    ops["own_inn"] = ops["company_id"].map(company_info["company_inn"])
    ops["company_industry"] = ops["company_id"].map(company_info["industry"])
    ops["role"] = ops["counterparty_name"].map(normalize_role)
    ops["industry"] = [
        taxonomy.industry_of(role, direction)
        for role, direction in zip(ops["role"], ops["direction"])
    ]
    ops["is_self"] = ops["counterparty_inn"].notna() & (
        ops["counterparty_inn"] == ops["own_inn"]
    )
    ops["purpose"] = ops["payment_purpose"].map(_text)

    result = ops.apply(classify_row, axis=1, result_type="expand")
    result.columns = ["category", "subcategory", "source", "confidence"]
    ops = pd.concat([ops, result], axis=1)

    ops = _smooth(ops)
    return ops


def _smooth(ops: pd.DataFrame) -> pd.DataFrame:
    """Доминирующая категория контрагента исправляет единичные выбросы."""
    eligible = (
        (ops["direction"] == "Списание")
        & ops["counterparty_inn"].notna()
        & (~ops["industry"].isin(taxonomy.NO_SMOOTHING))
        & (~ops["category"].isin(NO_SMOOTH_CATEGORIES))
    )
    subset = ops[eligible]

    keys = ["company_id", "counterparty_inn", "direction"]
    counts = (
        subset.groupby(keys + ["category"], dropna=False)
        .size()
        .rename("category_ops")
        .reset_index()
    )
    totals = counts.groupby(keys, dropna=False)["category_ops"].transform("sum")
    counts["share"] = counts["category_ops"] / totals

    dominant = (
        counts.sort_values(keys + ["category_ops"], ascending=True)
        .drop_duplicates(subset=keys, keep="last")
        .rename(columns={"category": "dominant_category"})
    )
    dominant = dominant[
        (dominant["category_ops"] >= SMOOTH_MIN_OPS) & (dominant["share"] >= SMOOTH_MIN_SHARE)
    ][keys + ["dominant_category"]]

    # Доминирующая подкатегория внутри доминирующей категории
    sub_counts = (
        subset.groupby(keys + ["category", "subcategory"], dropna=False)
        .size()
        .rename("n")
        .reset_index()
        .sort_values("n")
        .drop_duplicates(subset=keys + ["category"], keep="last")
    )
    sub_counts = sub_counts.rename(
        columns={"category": "dominant_category", "subcategory": "dominant_subcategory"}
    )[keys + ["dominant_category", "dominant_subcategory"]]

    dominant = dominant.merge(sub_counts, on=keys + ["dominant_category"], how="left")

    merged = ops.merge(dominant, on=keys, how="left")
    mask = (
        merged["dominant_category"].notna()
        & (merged["category"] != merged["dominant_category"])
    )
    merged.loc[mask, "category"] = merged.loc[mask, "dominant_category"]
    merged.loc[mask, "subcategory"] = merged.loc[mask, "dominant_subcategory"].fillna(
        merged.loc[mask, "subcategory"]
    )
    merged.loc[mask, "source"] = "smoothed"
    merged.loc[mask, "confidence"] = 0.85
    return merged.drop(columns=["dominant_category", "dominant_subcategory"])


def _money(value: float) -> str:
    return f"{value:,.0f}".replace(",", " ")


def report(ops: pd.DataFrame, company_id: str | None = None) -> None:
    total_ops = len(ops)
    total_amount = ops["amount"].sum()
    print(f"операций: {total_ops:,} | сумма: {_money(total_amount)} руб.".replace(",", " "))

    by_cat = (
        ops.groupby("category")
        .agg(ops=("amount", "size"), amount=("amount", "sum"))
        .assign(
            share_ops=lambda d: d.ops / total_ops,
            share_amount=lambda d: d.amount / total_amount,
        )
        .sort_values("amount", ascending=False)
    )
    print("\nкатегории:")
    print(f"{'категория':24} {'операций':>9} {'сумма, млн':>12} {'доля оп':>8} {'доля денег':>11}")
    for name, row in by_cat.iterrows():
        print(
            f"{name:24} {row.ops:>9,.0f} {row.amount / 1e6:>12,.1f} "
            f"{row.share_ops:>7.1%} {row.share_amount:>10.1%}"
        )

    other = by_cat.loc["Прочее"] if "Прочее" in by_cat.index else None
    if other is not None:
        print(
            f"\n«Прочее»: {other.share_ops:.1%} операций, "
            f"{other.share_amount:.1%} денег"
        )
    else:
        print("\n«Прочее»: 0.0% операций, 0.0% денег")

    print("\nисточник решения:")
    for source, count in ops["source"].value_counts().items():
        print(f"  {source:12} {count:>8,} ({count / total_ops:.1%})")

    print("\nподкатегории выручки:")
    revenue = ops[ops.category == "Выручка"]
    for name, count in revenue.subcategory.value_counts().items():
        amount = revenue.loc[revenue.subcategory == name, "amount"].sum()
        print(f"  {name:24} {count:>7,} операций, {amount / 1e6:>9,.1f} млн")

    taxes = ops[ops.category == "Налоги"]
    if not taxes.empty:
        print("\nподкатегории налогов:")
        for name, count in taxes.subcategory.value_counts().items():
            print(f"  {name:24} {count:>7,}")

    internal = ops[ops.category == "Внутренние и владелец"]
    if not internal.empty:
        print(
            f"\nвнутренние и владелец: {len(internal):,} операций, "
            f"{_money(internal.amount.sum())} руб."
        )
        for name, count in internal.subcategory.value_counts().items():
            amount = internal.loc[internal.subcategory == name, "amount"].sum()
            print(f"  {name:24} {count:>7,} операций, {_money(amount)} руб.")

    print("\nтоп-3 контрагента в категориях:")
    for category in by_cat.index[:8]:
        subset = ops[ops.category == category]
        top = (
            subset.groupby("counterparty_name")
            .amount.sum()
            .nlargest(3)
        )
        names = "; ".join(f"{name} ({_money(value)})" for name, value in top.items())
        print(f"  {category:24} {names}")
    print(
        "  (ПАО «Банк» в выручке и зарплате — это канал: эквайринг/СБП и "
        "зарплатный реестр, а не конечный плательщик или получатель)"
    )

    consistency = _stable_share(ops)
    smoothed = int((ops["source"] == "smoothed").sum())
    print(
        f"\nсглаживание по контрагентам: изменено {smoothed:,} операций; "
        f"доля контрагентов с одной категорией — {consistency:.1%}"
    )

    unknown_roles = (
        ops[ops.source == "fallback"]
        .groupby("role")
        .size()
        .sort_values(ascending=False)
    )
    if not unknown_roles.empty:
        print(f"\nнераспознанные роли ({len(unknown_roles)}):")
        for role, count in unknown_roles.head(10).items():
            print(f"  {role:40} {count:>6,}")

    if company_id:
        _report_company(ops, company_id)


def _stable_share(ops: pd.DataFrame) -> float:
    eligible = (
        (ops["direction"] == "Списание")
        & ops["counterparty_inn"].notna()
        & (~ops["industry"].isin(taxonomy.NO_SMOOTHING))
        & (~ops["category"].isin(NO_SMOOTH_CATEGORIES))
    )
    subset = ops[eligible]
    keys = ["company_id", "counterparty_inn", "direction"]
    counts = (
        subset.groupby(keys + ["category"], dropna=False)
        .size()
        .rename("n")
        .reset_index()
    )
    totals = counts.groupby(keys, dropna=False)["n"].transform("sum")
    counts["share"] = counts["n"] / totals
    max_share = counts.groupby(keys, dropna=False)["share"].max()
    sizes = counts.groupby(keys, dropna=False)["n"].sum()
    stable = max_share[sizes >= SMOOTH_MIN_OPS]
    return float((stable >= 0.95).mean()) if len(stable) else 0.0


def _report_company(ops: pd.DataFrame, company_id: str) -> None:
    company = ops[ops.company_id == company_id]
    if company.empty:
        print(f"\nкомпания {company_id} не найдена")
        return
    total = company.amount.sum()
    print(f"\n=== {company_id}: {len(company):,} операций, {_money(total)} руб. ===")
    table = (
        company.groupby("category")
        .agg(ops=("amount", "size"), amount=("amount", "sum"))
        .assign(share=lambda d: d.amount / total)
        .sort_values("amount", ascending=False)
    )
    for name, row in table.iterrows():
        print(
            f"  {name:24} {row.ops:>6,.0f} операций {row.amount / 1e6:>9,.1f} млн "
            f"({row.share:>5.1%})"
        )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--operations", type=Path, default=Path("data/all_operations.csv"))
    parser.add_argument("--companies", type=Path, default=Path("data/companies.csv"))
    parser.add_argument("--company", default=None, help="показать разрез по компании")
    parser.add_argument(
        "--save", type=Path, default=Path("artifacts/analytics/operations_categorized.csv")
    )
    args = parser.parse_args()

    operations = pd.read_csv(
        args.operations,
        sep=";",
        encoding="utf-8-sig",
        dtype={"company_id": str, "counterparty_inn": str},
        parse_dates=["operation_date"],
    )
    companies = pd.read_csv(
        args.companies, sep=";", encoding="utf-8-sig", dtype={"company_inn": str, "company_id": str}
    )

    categorized = classify(operations, companies)
    report(categorized, company_id=args.company)

    if args.save:
        args.save.parent.mkdir(parents=True, exist_ok=True)
        columns = [
            "company_id", "operation_id", "operation_date", "amount", "direction",
            "counterparty_inn", "counterparty_name", "role", "industry",
            "category", "subcategory", "source", "confidence",
        ]
        categorized[columns].to_csv(args.save, index=False, sep=";", encoding="utf-8-sig")
        print(f"\nсохранено: {args.save}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
