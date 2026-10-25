import { useEffect, useState } from 'react';
import { Link, useLocation, useParams } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import '../App.css';
import FinanceChart from '../components/FinanceChart';
import { toast } from 'sonner';
import { addOperation, calculateScenario, getForecast, processForecast, uploadStatement, ForecastAlert, ForecastResponse, ScheduledPayment, StatementOperation, rubles } from '../api/forecast';
import { getPulse, renamePulse } from '../api/pulses';

const Dashboard = () => {
    const { pulseId = '' } = useParams();
    const location = useLocation();
    const { user, loading, openModal } = useAuth();
    const [isLoading, setIsLoading] = useState(false);
    const [pulseName, setPulseName] = useState('Остаток');
    const [forecast, setForecast] = useState<ForecastResponse | null>(null);
    const [showBounds, setShowBounds] = useState(false);
    const [operationOpen, setOperationOpen] = useState(false);
    const [operationDate, setOperationDate] = useState('');
    const [operationAmount, setOperationAmount] = useState('');
    const [operationIncome, setOperationIncome] = useState(false);
    const [operationNote, setOperationNote] = useState('');
    const [operationRecurring, setOperationRecurring] = useState(false);
    const [scenarioMode, setScenarioMode] = useState(false);
    const [scenarioDate, setScenarioDate] = useState('');
    const [scenarioAmount, setScenarioAmount] = useState('');
    const [scenarioIncome, setScenarioIncome] = useState(false);
    const [scenarioResult, setScenarioResult] = useState<ForecastResponse | null>(null);

    const recurringPayments = (data: ForecastResponse | null): ScheduledPayment[] =>
        data?.recurring_payments || [];

    const recurringByCategory = (payments: ScheduledPayment[]) => {
        const groups = new Map<string, { category: string; is_income: boolean; count: number; amount: number }>();
        payments.forEach((payment) => {
            const category = payment.category || 'Без категории';
            const key = `${category}:${payment.is_income}`;
            const group = groups.get(key) || { category, is_income: payment.is_income, count: 0, amount: 0 };
            group.count += 1;
            group.amount += payment.amount;
            groups.set(key, group);
        });
        return [...groups.values()];
    };

    const nearestGap = (alerts: ForecastAlert[] | undefined): ForecastAlert | null => {
        if (!alerts?.length) {
            return null;
        }
        return [...alerts].sort((left, right) => left.date.localeCompare(right.date))[0];
    };

    const chancePercent = (chance: number) => Math.round(Math.min(1, Math.max(0, chance)) * 100);

    const chanceWord = (chance: number) => {
        const percent = chancePercent(chance);
        if (percent >= 70) {
            return `шанс ${percent}%, почти наверняка`;
        }
        if (percent >= 30) {
            return `шанс ${percent}%`;
        }
        return `шанс небольшой, ${percent}%`;
    };

    const remember = (next: ForecastResponse): ForecastResponse => ({
        ...next,
        days: next.days || [],
        alerts: next.alerts || [],
        operations: next.operations || [],
        history: next.history || [],
        rare_payments: next.rare_payments || [],
    });

    const showForecast = (next: ForecastResponse) => {
        setForecast(remember(next));
        setShowBounds(false);
        setIsLoading(false);
    };

    // Training finishes in a few seconds. Keep asking until the new line is ready.
    const pullForecast = async () => {
        const deadline = Date.now() + 25000;
        let next = await processForecast(pulseId);
        while (Date.now() < deadline) {
            const status = next.training?.status;
            const ready = (next.days?.length || 0) > 0 && status !== 'queued' && status !== 'running';
            if (status === 'failed' || ready) break;
            await new Promise((resolve) => setTimeout(resolve, 1000));
            next = await processForecast(pulseId);
        }
        const ready = (next.days?.length || 0) > 0 && next.training?.status !== 'failed';
        setForecast(remember(next));
        setShowBounds(ready);
        return next;
    };

    const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>, append = false) => {
        const file = e.target.files?.[0];
        e.target.value = '';
        if (!file) return;
        setIsLoading(true);
        try {
            const saved = await uploadStatement(pulseId, file, append);
            if (saved.training?.status === 'failed') {
                showForecast(saved);
                toast.error(saved.training.error || 'Модель не удалось поставить на обучение');
                return;
            }
            const next = await pullForecast();
            setIsLoading(false);
            if (next.training?.status === 'failed') {
                toast.error(next.training.error || 'Обучение не удалось');
                return;
            }
            if (!(next.days?.length || 0)) {
                toast.message('Модель ещё учится. Нажмите «Обработать» ещё раз.');
                return;
            }
            toast.success('Выписка загружена, прогноз обновлён');
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Ошибка загрузки выписки');
            setIsLoading(false);
        }
    };

    const handleProcess = async () => {
        setIsLoading(true);
        try {
            const next = await pullForecast();
            setIsLoading(false);
            if (next.training?.status === 'failed') {
                toast.error(next.training.error || 'Обучение не удалось');
                return;
            }
            if (!(next.days?.length || 0)) {
                toast.message('Модель ещё учится. Нажмите «Обработать» ещё раз.');
                return;
            }
            toast.success('Прогноз на 30 дней готов');
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Прогноз ещё не готов');
            setIsLoading(false);
        }
    };

    // Загрузка демо-данных из public/data/financial_data.csv
    const loadDemoData = async () => {
        setIsLoading(true);
        try {
            const response = await fetch('/data/financial_data.csv');
            if (!response.ok) throw new Error('Демо-файл не найден');
            const file = new File([await response.blob()], 'financial_data.csv', { type: 'text/csv' });
            showForecast(await uploadStatement(pulseId, file));
        } catch (err) {
            toast.error('Не удалось загрузить демо-файл. Проверьте папку public/data');
            setIsLoading(false);
        }
    };

    // Сброс к начальному экрану
    const handleReset = () => {
        setForecast(null);
        setShowBounds(false);
    };

    const statementEnd = forecast?.history?.length
        ? forecast.history[forecast.history.length - 1].date
        : '';
    const operationIsPlan = Boolean(operationDate && statementEnd && operationDate > statementEnd);

    const handleAddOperation = async (event: React.FormEvent) => {
        event.preventDefault();
        const rublesAmount = Number(operationAmount.replace(',', '.'));
        if (!operationDate || !Number.isFinite(rublesAmount) || rublesAmount <= 0) {
            toast.error('Укажите дату и сумму больше нуля');
            return;
        }
        setIsLoading(true);
        try {
            await addOperation(pulseId, {
                date: operationDate,
                amount: Math.round(rublesAmount * 100),
                is_income: operationIncome,
                description: operationNote.trim(),
                recurring: operationIsPlan && operationRecurring,
            });
            const next = await pullForecast();
            setOperationOpen(false);
            setOperationAmount('');
            setOperationNote('');
            setOperationRecurring(false);
            setIsLoading(false);
            if (next.training?.status === 'failed') {
                toast.error(next.training.error || 'Операция сохранена, но модель не переобучилась');
                return;
            }
            toast.success(operationIsPlan ? 'Платёж учтён, прогноз пересчитан' : 'Операция учтена, прогноз пересчитан');
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Не удалось сохранить операцию');
            setIsLoading(false);
        }
    };

    const handleChartDateClick = (date: string) => {
        if (scenarioMode) {
            setScenarioDate(date);
            return;
        }
        setOperationDate(date);
        setOperationOpen(true);
        window.requestAnimationFrame(() => {
            document.querySelector('.GridAdd')?.scrollIntoView({ behavior: 'smooth', block: 'center' });
        });
    };

    const runScenario = async (event: React.FormEvent) => {
        event.preventDefault();
        const amount = Number(scenarioAmount.replace(',', '.'));
        if (!scenarioDate || !Number.isFinite(amount) || amount <= 0) {
            toast.error('Выберите дату и укажите сумму больше нуля');
            return;
        }
        setScenarioResult(null);
        setIsLoading(true);
        try {
            const result = await calculateScenario(pulseId, [{
                date: scenarioDate,
                amount: Math.round(amount * 100),
                is_income: scenarioIncome,
                category: 'Сценарий',
            }]);
            setScenarioResult(result);
            toast.success('Prophet переобучен на временном сценарии');
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Не удалось переобучить модель');
        } finally {
            setIsLoading(false);
        }
    };

    const chartForecast = scenarioResult || forecast;
    const chartShowBounds = showBounds || Boolean(scenarioResult);

    useEffect(() => {
        if (!user || !pulseId) return;
        let stop = false;
        const fresh = Boolean((location.state as { fresh?: boolean } | null)?.fresh);
        (async () => {
            setIsLoading(true);
            try {
                const pulse = await getPulse(pulseId);
                if (stop) return;
                setPulseName(pulse.name);
                const loaded = await getForecast(pulseId);
                if (stop) return;
                if (fresh && !(loaded.days?.length)) {
                    await pullForecast();
                    return;
                }
                setForecast(remember(loaded));
                setShowBounds((loaded.days?.length || 0) > 0);
                setIsLoading(false);
            } catch (err) {
                if (!stop) {
                    toast.error(err instanceof Error ? err.message : 'Не удалось открыть пульс');
                    setIsLoading(false);
                }
            }
        })();
        return () => { stop = true; };
    }, [user?.id, pulseId]);

    return (
        <div className="DashboardContainer">

            {/* ШАПКА (Приветствие слева, кнопка/статус справа) */}
            <div className="DashboardHeader">
                <div>
                    <h1>{pulseName}</h1>
                    <p><Link to="/">Все пульсы</Link>. Прогноз на 30 дней.</p>
                </div>
                {!!forecast?.days?.length && (
                    <div className="UploadActions">
                        <button className="Button" type="button" onClick={handleProcess}>
                            Обработать
                        </button>
                        <label className="Button upload-btn">
                            Объединить CSV
                            <input type="file" accept=".csv" onChange={(event) => handleFileUpload(event, true)} hidden />
                        </label>
                        <button className="Button" onClick={handleReset}>
                            Загрузить другой файл
                        </button>
                    </div>
                )}
            </div>

            {!user && !loading && (
                <div className="GuestPulse" aria-hidden="true">
                    <div className="skeleton skeleton-title" />
                    <div className="skeleton skeleton-chart" />
                    <div className="skeleton skeleton-text" />
                    <div className="skeleton skeleton-text" />
                    <button className="Button" onClick={openModal}>Войти</button>
                </div>
            )}

            {/* ЗОНА ЗАГРУЗКИ */}
            {user && !forecast?.history?.length && !forecast?.days?.length && !isLoading && (
                <div className="UploadZone">
                    <label className="PulseName">
                        Имя
                        <input
                            type="text"
                            maxLength={80}
                            value={pulseName}
                            onChange={(event) => setPulseName(event.target.value)}
                            onBlur={() => {
                                const title = pulseName.trim();
                                if (!title || !pulseId) return;
                                renamePulse(pulseId, title).catch((err) => {
                                    toast.error(err instanceof Error ? err.message : 'Не удалось сохранить имя');
                                });
                            }}
                        />
                    </label>
                    <h2>Загрузите выписку</h2>
                    <p>CSV из банка или 1С. Прогноз построится после загрузки.</p>

                    <div className="UploadActions">
                        <label className="Button upload-btn">
                            Выбрать файл .CSV
                            <input type="file" accept=".csv" onChange={handleFileUpload} hidden />
                        </label>

                        <button className="Button" onClick={loadDemoData}>
                            Попробовать на демо-данных
                        </button>
                    </div>
                </div>
            )}

            {/* ЛОАДЕР */}
            {isLoading && (
                <div className="LoadingZone">
                    <div className="spinner"></div>
                    <h2>Считаем прогноз</h2>
                    <p>Обычно это занимает несколько секунд</p>
                </div>
            )}

            {/* РЕЗУЛЬТАТЫ (Сетка) */}
            {(forecast?.history?.length || forecast?.days?.length) && !isLoading ? (
                <div className="ResultsGrid">

                    {/* Карточка бизнеса (Правая колонка, верх) */}
                    <div className="GridAdd">
                        <div className="ChartHead">
                            <div>
                                <h3 className="SectionTitle">Своя операция</h3>
                                <p className="OperationHint">Нажмите на график — дата подставится автоматически.</p>
                            </div>
                        </div>
                        <form className="OperationForm" onSubmit={handleAddOperation}>
                            <label>
                                Дата
                                <input type="date" value={operationDate} onChange={(event) => setOperationDate(event.target.value)} required />
                            </label>
                            <label>
                                Сумма, ₽
                                <input
                                    type="number"
                                    min="0.01"
                                    step="0.01"
                                    value={operationAmount}
                                    onChange={(event) => setOperationAmount(event.target.value)}
                                    placeholder="10000"
                                    required
                                />
                            </label>
                            <label>
                                Комментарий
                                <input
                                    type="text"
                                    value={operationNote}
                                    onChange={(event) => setOperationNote(event.target.value)}
                                    placeholder="Назначение платежа"
                                />
                            </label>
                            <div className="OperationForm-choice">
                                <button type="button" className={!operationIncome ? 'is-active operation-expense' : 'operation-expense'} onClick={() => setOperationIncome(false)}>↓ Расход</button>
                                <button type="button" className={operationIncome ? 'is-active operation-income' : 'operation-income'} onClick={() => setOperationIncome(true)}>↑ Приход</button>
                            </div>
                            {operationIsPlan && (
                                <label className="OperationForm-check">
                                    <input
                                        type="checkbox"
                                        checked={operationRecurring}
                                        onChange={(event) => setOperationRecurring(event.target.checked)}
                                    />
                                    Повторять каждый месяц
                                </label>
                            )}
                            <p className="GapCard-note">
                                {operationIsPlan
                                    ? 'Дата позже выписки: платёж сдвинет прогноз и не изменит фактический остаток.'
                                    : 'Дата в пределах выписки: платёж сдвинет фактический остаток и не станет ежедневным расходом.'}
                            </p>
                            <button className="Button" type="submit">Сохранить и пересчитать</button>
                        </form>
                    </div>

                    {/* График (На всю ширину, или по центру) */}
                    <div className="GridChart">
                        {/* Умный баннер (Перенесен над графиком для логики) */}
                        {showBounds && forecast.rough && <p>Прогноз ориентировочный: истории меньше 30 дней.</p>}
                        {showBounds && nearestGap(forecast.alerts) && (
                            <div className="SuccessBanner" style={{
                                borderColor: 'var(--error)',
                                color: 'var(--error)',
                                background: 'rgba(244, 63, 94, 0.1)',
                                marginBottom: '20px',
                                width: '100%'
                            }}>
                                Ближайший разрыв {nearestGap(forecast.alerts)?.date}: не хватает около {rubles(nearestGap(forecast.alerts)?.shortfall || 0).toLocaleString('ru-RU')} ₽, {chanceWord(nearestGap(forecast.alerts)?.chance || 0)}.
                            </div>
                        )}
                        {showBounds && !forecast.alerts?.length && (
                            <div className="SuccessBanner" style={{ marginBottom: '20px', width: '100%' }}>
                                Кассовых разрывов на 30 дней не видно.
                            </div>
                        )}

                        <FinanceChart
                            showBounds={chartShowBounds}
                            onDateClick={handleChartDateClick}
                            history={(forecast.history?.length
                                ? forecast.history
                                : [{ date: forecast.days[0].date, balance: forecast.days[0].expected_balance }]
                            ).map((point) => ({
                                date: point.date,
                                balance: rubles(point.balance),
                            }))}
                            forecast={chartForecast.days.map((day) => ({
                                date: day.date,
                                balance: rubles(day.expected_balance),
                                low: rubles(day.balance_low),
                                high: rubles(day.balance_high),
                            }))}
                        />
                        <div className="ScenarioTools">
                            <button className={scenarioMode ? 'Button is-active' : 'Button'} type="button" onClick={() => { setScenarioMode((value) => !value); setScenarioDate(''); setScenarioResult(null); }}>
                                {scenarioMode ? 'Закрыть сценарий' : 'Что если?'}
                            </button>
                            {scenarioMode && <form className="ScenarioEditor" onSubmit={runScenario}>
                                <p className="ScenarioHint">Нажмите на будущую дату прогноза. Выбрано: <strong>{scenarioDate || '—'}</strong></p>
                                <div className="ScenarioRow">
                                    <input type="date" value={scenarioDate} onChange={(event) => setScenarioDate(event.target.value)} />
                                    <input value={scenarioAmount} onChange={(event) => setScenarioAmount(event.target.value)} placeholder="Сумма, ₽" inputMode="decimal" />
                                    <button className="Button" type="button" onClick={() => setScenarioIncome((value) => !value)}>{scenarioIncome ? '↑ Приход' : '↓ Расход'}</button>
                                </div>
                                <button className="Button upload-btn" type="submit" disabled={isLoading}>{isLoading ? 'Переобучение Prophet…' : 'Перестроить прогноз'}</button>
                            </form>}
                        </div>
                    </div>

                    <div className="GridOps">
                        <h3 className="SectionTitle">Последние операции</h3>
                        {(forecast.operations || []).length ? (
                            <div className="GapList OperationList">
                                {(forecast.operations || []).slice(0, 10).map((op: StatementOperation, index) => (
                                    <div key={`${op.date}:${op.description}:${index}`} className="GapCard OperationCard">
                                        <div className="OperationCard-main">
                                            <div className="GapCard-date">{op.date}</div>
                                            <div className="GapCard-note">
                                                {op.manual ? 'Добавлено вручную' : (op.category || op.description || 'Без категории')}
                                                {op.manual && op.description ? ` · ${op.description}` : ''}
                                            </div>
                                        </div>
                                        <div className="GapCard-sum">
                                            <div className={op.is_income ? 'Amount--plus' : 'Amount--minus'}>
                                                {op.is_income ? '+' : '−'}{rubles(Math.abs(op.amount)).toLocaleString('ru-RU')} ₽
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        ) : (
                            <p className="GapCard-note">Операций в выписке пока нет.</p>
                        )}
                    </div>

                    {/* Предупреждения (Снизу на всю ширину) */}
                    {showBounds && !!forecast.alerts?.length && (
                        <div className="GridWarnings">
                            <h3 className="SectionTitle">Предупреждения о разрывах:</h3>
                            <div className="GapList">
                                {forecast.alerts.map((alert) => (
                                    <div key={alert.date + alert.reason} className="GapCard">
                                        <div>
                                            <div className="GapCard-date">
                                                {alert.date}
                                            </div>
                                            <div className="GapCard-note">
                                                {alert.reason}, шанс {chancePercent(alert.chance)}%
                                            </div>
                                        </div>
                                        <div className="GapCard-sum">
                                            <div className="GapCard-amount">
                                                -{rubles(alert.shortfall).toLocaleString('ru-RU')} ₽
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                </div>
            ) : null}
        </div>
    );
};

export default Dashboard;
