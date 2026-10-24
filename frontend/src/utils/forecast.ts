// utils/forecast.ts
// ПРАВИЛЬНЫЙ ИМПОРТ (когда создадите csv.ts):
import { Transaction } from './csv';

export interface ForecastPoint { date: string; balance: number; }
export interface CashGap {
    date: string;
    shortage: number;      // сколько не хватает
    probability: number;   // 0..1 (грубая оценка)
}

export function forecastBalance(
    transactions: Transaction[],
    currentBalance: number,
    days = 30
): { forecast: ForecastPoint[]; gaps: CashGap[] } {
    // 1. Группируем транзакции по категориям
    const byCategory = new Map<string, Transaction[]>();
    for (const t of transactions) {
        if (!byCategory.has(t.category)) byCategory.set(t.category, []);
        byCategory.get(t.category)!.push(t);
    }

    // 2. Для каждой категории считаем средний интервал и сумму
    const recurring: { category: string; intervalDays: number; avgAmount: number }[] = [];
    for (const [category, items] of byCategory) {
        const dates = items.map(t => new Date(t.date).getTime()).sort((a, b) => a - b);
        if (dates.length < 2) continue;
        const gaps = dates.slice(1).map((d, i) => d - dates[i]);
        const interval = gaps.reduce((a, b) => a + b, 0) / gaps.length / 86400000;
        const avg = items.reduce((s, t) => s + t.amount, 0) / items.length;
        // «Регулярность»: если интервал < 45 дней — считаем регулярным
        if (interval > 0 && interval < 45) {
            recurring.push({ category, intervalDays: interval, avgAmount: avg });
        }
    }

    // 3. Строим прогноз на days дней вперёд
    const forecast: ForecastPoint[] = [];
    const start = new Date();
    let balance = currentBalance;
    const lastDateByCategory = new Map<string, Date>();
    for (const t of transactions) {
        const d = new Date(t.date);
        const prev = lastDateByCategory.get(t.category);
        if (!prev || d > prev) lastDateByCategory.set(t.category, d);
    }

    for (let i = 1; i <= days; i++) {
        const day = new Date(start);
        day.setDate(day.getDate() + i);

        for (const r of recurring) {
            const last = lastDateByCategory.get(r.category);
            if (!last) continue;
            const daysSinceLast = Math.floor((day.getTime() - last.getTime()) / 86400000);
            const expectedOffset = Math.floor(r.intervalDays);
            if (expectedOffset > 0 && daysSinceLast > 0 &&
                daysSinceLast % expectedOffset === 0) {
                balance += r.avgAmount;
            }
        }

        forecast.push({
            date: day.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' }),
            balance: Math.round(balance),
        });
    }

    // 4. Ищем кассовые разрывы (баланс < 0)
    const gaps: CashGap[] = forecast
        .filter(p => p.balance < 0)
        .map(p => ({
            date: p.date,
            shortage: -p.balance,
            // TODO: заменить заглушку на модель. В UI не показывать.
            probability: 0.75,
        }));

    return { forecast, gaps };
}