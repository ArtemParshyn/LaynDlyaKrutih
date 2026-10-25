import { useEffect, useRef, useState } from 'react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, ResponsiveContainer, Legend, Tooltip, ReferenceArea, ReferenceLine } from 'recharts';
import '../App.css';

interface ChartPoint {
    date: string;
    balance?: number;
    low?: number;
    high?: number;
    mid?: number;
    scenario?: number;
}

interface FinanceChartProps {
    history: ChartPoint[];
    forecast: ChartPoint[];
    showBounds?: boolean;
    onDateClick?: (date: string) => void;
    scenarioForecast?: ChartPoint[];
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

// The visible line is the fact on the left and the expected forecast on the right.
const levelOf = (point: ChartPoint) =>
    point.balance != null ? point.balance : point.mid;

const FinanceChart = ({ history, forecast, showBounds = false, onDateClick, scenarioForecast = [] }: FinanceChartProps) => {
    const [windowDays, setWindowDays] = useState(90);
    const [dragFrom, setDragFrom] = useState<string | null>(null);
    const [dragTo, setDragTo] = useState<string | null>(null);
    const draggedRef = useRef(false);
    const safeHistory = (history || []).filter((point) => point?.date && Number.isFinite(point.balance));
    const safeForecast = (forecast || []).filter((point) => point?.date && Number.isFinite(point.balance));
    const last = safeHistory[safeHistory.length - 1];
    const visibleHistory = safeHistory.slice(-windowDays);
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

    // Forecast stays on mid/low/high. Putting it on balance draws the fact
    // series over the expected line, so a wavy Prophet forecast looks flat.
    if (showBounds && safeForecast.length) {
        safeForecast.forEach((point) => points.push({
            date: point.date,
            low: point.low,
            high: point.high,
            mid: point.balance,
        }));
    }
    const scenarioByDate = new Map(scenarioForecast.map((point) => [point.date, point.balance]));
    points.forEach((point) => { point.scenario = scenarioByDate.get(point.date); });

    const forecastValues = safeForecast.flatMap((point) => [point.low, point.high, point.balance])
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

    useEffect(() => {
        setDragFrom(null);
        setDragTo(null);
    }, [windowDays, showBounds, points.length]);

    const byDate = new Map(points.map((point, index) => [point.date, index]));
    const anchorDate = dragFrom;
    const endDate = dragTo;
    const from = anchorDate != null ? points[byDate.get(anchorDate) ?? -1] : points[0];
    const to = endDate != null ? points[byDate.get(endDate) ?? -1] : points[points.length - 1];
    const bandFrom = anchorDate && endDate && (byDate.get(anchorDate) ?? 0) <= (byDate.get(endDate) ?? 0)
        ? anchorDate : endDate;
    const bandTo = anchorDate && endDate && (byDate.get(anchorDate) ?? 0) <= (byDate.get(endDate) ?? 0)
        ? endDate : anchorDate;
    const fromValue = from ? levelOf(from) : undefined;
    const toValue = to ? levelOf(to) : undefined;
    const change = fromValue != null && toValue != null && fromValue !== 0
        ? ((toValue - fromValue) / Math.abs(fromValue)) * 100
        : null;
    const delta = fromValue != null && toValue != null ? toValue - fromValue : null;
    const changeText = change == null
        ? 'Зажмите мышь на графике и протяните период'
        : `${delta != null && delta > 0 ? '+' : ''}${delta != null ? Math.round(delta).toLocaleString('ru-RU') : ''} ₽ (${change > 0 ? '+' : ''}${change.toLocaleString('ru-RU', { maximumFractionDigits: 1 })}%)`;

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
            {points.length > 1 && (
                <p className={change != null && change < 0 ? 'ChartChange ChartChange--down' : 'ChartChange'}>
                    {from?.date && to?.date ? `${dayLabel(from.date)} → ${dayLabel(to.date)}: ` : ''}
                    <strong>{changeText}</strong>
                </p>
            )}

            {points.length === 0 ? (
                <p className="GapCard-note">Недостаточно точек, чтобы построить график.</p>
            ) : null}
            {points.length > 0 && (
            <ResponsiveContainer width="100%" height={340}>
                <AreaChart
                    data={points}
                    margin={{ top: 10, right: 16, left: 8, bottom: 0 }}
                    onMouseDown={(state) => {
                        const date = state?.activeLabel ? String(state.activeLabel) : null;
                        if (!date) return;
                        draggedRef.current = false;
                        setDragFrom(date);
                        setDragTo(date);
                    }}
                    onMouseMove={(state) => {
                        if (!dragFrom) return;
                        const date = state?.activeLabel ? String(state.activeLabel) : null;
                        if (date) {
                            if (date !== dragFrom) draggedRef.current = true;
                            setDragTo(date);
                        }
                    }}
                    onMouseUp={() => {
                        setDragFrom(null);
                        setDragTo(null);
                    }}
                    onMouseLeave={() => {
                        if (dragFrom) {
                            setDragFrom(null);
                            setDragTo(null);
                        }
                    }}
                    onClick={(state) => {
                        const date = state?.activeLabel ? String(state.activeLabel) : null;
                        if (date && !draggedRef.current) onDateClick?.(date);
                        draggedRef.current = false;
                    }}
                >
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
                            fullMoney(Number(value ?? 0)),
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
                    {!!scenarioForecast.length && (
                        <Area type="monotone" dataKey="scenario" name="scenario" stroke="#c5304b" strokeWidth={3} strokeDasharray="7 5" fill="transparent" connectNulls={false} activeDot={{ r: 4 }} />
                    )}
                    {showBounds && (
                        <Area type="monotone" dataKey="low" name="low" stroke="#c5304b" strokeDasharray="4 4" strokeWidth={3.5} fill="transparent" connectNulls={false} activeDot={{ r: 4 }} />
                    )}
                    {showBounds && (
                        <Area type="monotone" dataKey="high" name="high" stroke="#7f9aa8" strokeWidth={1.5} fill="transparent" connectNulls={false} activeDot={{ r: 3 }} />
                    )}
                    {showBounds && (
                        <Area type="monotone" dataKey="mid" name="mid" stroke="#d7c07a" strokeWidth={2} fill="transparent" connectNulls={false} activeDot={{ r: 3 }} />
                    )}
                    <Area type="monotone" dataKey="balance" name="balance" stroke="var(--accent)" strokeWidth={3} fill="url(#colorBalance)" connectNulls={false} activeDot={{ r: 5 }} />
                    {bandFrom && bandTo && bandFrom !== bandTo && (
                        <ReferenceArea
                            x1={bandFrom}
                            x2={bandTo}
                            fill={change != null && change < 0 ? 'rgba(244, 63, 94, 0.28)' : 'rgba(0, 135, 74, 0.28)'}
                            strokeOpacity={0}
                        />
                    )}
                    {bandFrom && bandTo && bandFrom !== bandTo && (
                        <ReferenceLine x={bandFrom} stroke={change != null && change < 0 ? '#fb7185' : '#00874A'} strokeDasharray="4 4" />
                    )}
                    {bandFrom && bandTo && bandFrom !== bandTo && (
                        <ReferenceLine x={bandTo} stroke={change != null && change < 0 ? '#fb7185' : '#00874A'} strokeDasharray="4 4" />
                    )}
                </AreaChart>
            </ResponsiveContainer>
            )}
        </div>
    );
};

export default FinanceChart;
