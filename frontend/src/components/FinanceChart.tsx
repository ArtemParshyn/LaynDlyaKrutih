import { useState } from 'react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, ResponsiveContainer, Legend, Tooltip, Brush } from 'recharts';
import '../App.css';

interface ChartPoint {
    date: string;
    balance?: number;
    low?: number;
    high?: number;
    mid?: number;
}

interface FinanceChartProps {
    history: ChartPoint[];
    forecast: ChartPoint[];
    showBounds?: boolean;
}

const money = (value: number) => {
    const abs = Math.abs(value);
    const sign = value < 0 ? '−' : '';
    if (abs >= 1_000_000) {
        return `${sign}${(abs / 1_000_000).toLocaleString('ru-RU', { maximumFractionDigits: 1 })} млн`;
    }
    if (abs >= 10_000) {
        return `${sign}${Math.round(abs / 1000).toLocaleString('ru-RU')} тыс`;
    }
    return `${sign}${Math.round(abs).toLocaleString('ru-RU')}`;
};

const fullMoney = (value: number) =>
    `${Math.round(value).toLocaleString('ru-RU')} ₽`;

const dayLabel = (value: string) => {
    const [, month, day] = value.split('-');
    return day && month ? `${day}.${month}` : value;
};

const FinanceChart = ({ history, forecast, showBounds = false }: FinanceChartProps) => {
    const [windowDays, setWindowDays] = useState(90);
    const last = history[history.length - 1];
    const visibleHistory = history.slice(-windowDays);
    const points: ChartPoint[] = visibleHistory.map((point, index) => {
        const isJoin = showBounds && index === visibleHistory.length - 1;
        return {
            date: point.date,
            balance: point.balance,
            low: isJoin ? point.balance : undefined,
            high: isJoin ? point.balance : undefined,
            mid: isJoin ? point.balance : undefined,
        };
    });

    if (showBounds && last && forecast.length) {
        forecast.forEach((point) => points.push({
            date: point.date,
            low: point.low,
            high: point.high,
            mid: point.balance,
        }));
    }

    const forecastValues = forecast.flatMap((point) => [point.low, point.high, point.balance])
        .filter((value): value is number => value != null);
    const historyValues = visibleHistory.map((point) => point.balance).filter((value): value is number => value != null);
    const historySpan = historyValues.length
        ? Math.max(...historyValues) - Math.min(...historyValues)
        : 0;
    const forecastSpan = forecastValues.length
        ? Math.max(...forecastValues) - Math.min(...forecastValues)
        : 0;
    const zoomToForecast = showBounds && forecastSpan > 0 && forecastSpan < historySpan * 0.15;
    const pad = Math.max(forecastSpan * 0.35, 1);
    const yDomain = zoomToForecast
        ? [Math.min(...forecastValues, last?.balance ?? forecastValues[0]) - pad,
            Math.max(...forecastValues, last?.balance ?? forecastValues[0]) + pad]
        : undefined;

    return (
        <div className="FinanceChart">
            <div className="ChartHead">
                <h3 className="SectionTitle">{showBounds ? 'Прогноз с границами' : 'Остаток по выписке'}</h3>
                <div className="ChartRange">
                    {[30, 60, 90].map((days) => (
                        <button
                            key={days}
                            type="button"
                            className={windowDays === days ? 'ChartRange-button is-active' : 'ChartRange-button'}
                            onClick={() => setWindowDays(days)}
                        >
                            {days} дн
                        </button>
                    ))}
                </div>
            </div>

            <ResponsiveContainer width="100%" height={340}>
                <AreaChart data={points} margin={{ top: 10, right: 16, left: 8, bottom: 0 }}>
                    <defs>
                        <linearGradient id="colorBalance" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--accent)" stopOpacity={0.3}/>
                            <stop offset="95%" stopColor="var(--accent)" stopOpacity={0}/>
                        </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="var(--border-soft)" vertical={false} />
                    <XAxis
                        dataKey="date"
                        stroke="var(--text-muted)"
                        tickFormatter={dayLabel}
                        minTickGap={28}
                        tick={{ fontSize: 12 }}
                    />
                    <YAxis
                        stroke="var(--text-muted)"
                        domain={yDomain || ['auto', 'auto']}
                        allowDataOverflow
                        width={84}
                        tickFormatter={money}
                        tick={{ fontSize: 12 }}
                    />
                    <Tooltip
                        formatter={(value, name) => [
                            fullMoney(Number(value)),
                            name === 'balance' ? 'Факт'
                                : name === 'mid' ? 'Средний прогноз'
                                    : name === 'high' ? 'Верхняя граница'
                                        : 'Нижняя граница',
                        ]}
                        labelFormatter={(label) => String(label)}
                        contentStyle={{ background: 'var(--surface)', border: '1px solid var(--border)', borderRadius: 8 }}
                    />
                    {showBounds && (
                        <Legend
                            formatter={(value) => (
                                value === 'balance' ? 'Факт'
                                    : value === 'mid' ? 'Средний прогноз'
                                        : value === 'high' ? 'Верхняя граница'
                                            : 'Нижняя граница'
                            )}
                        />
                    )}
                    {showBounds && (
                        <Area type="monotone" dataKey="low" name="low" stroke="#fb7185" strokeDasharray="2 6" strokeWidth={2} fill="transparent" connectNulls={false} activeDot={{ r: 4 }} />
                    )}
                    {showBounds && (
                        <Area type="monotone" dataKey="high" name="high" stroke="#38bdf8" strokeWidth={3} fill="transparent" connectNulls={false} activeDot={{ r: 4 }} />
                    )}
                    {showBounds && (
                        <Area type="monotone" dataKey="mid" name="mid" stroke="#fbbf24" strokeWidth={3} fill="transparent" connectNulls={false} activeDot={{ r: 4 }} />
                    )}
                    <Area type="monotone" dataKey="balance" name="balance" stroke="var(--accent)" strokeWidth={3} fill="url(#colorBalance)" connectNulls={false} activeDot={{ r: 5 }} />
                    {points.length > 14 && (
                        <Brush dataKey="date" height={24} stroke="var(--accent)" tickFormatter={dayLabel} travellerWidth={8} />
                    )}
                </AreaChart>
            </ResponsiveContainer>
        </div>
    );
};

export default FinanceChart;
