import { useEffect, useState } from 'react';
import { useAuth } from '../context/AuthContext';
import '../App.css';
import FinanceChart from '../components/FinanceChart';
import { toast } from 'sonner';
import { getForecast, uploadStatement, ForecastAlert, ForecastResponse, ScheduledPayment, StatementOperation, rubles } from '../api/forecast';
import PlasticCard from '../components/BusinessCard/BuisnessCard';

const Dashboard = () => {
    const { user, loading, openModal } = useAuth();
    const [isLoading, setIsLoading] = useState(false);


    // Храним данные графика и найденные разрывы
    const storageKey = `forecast:${user?.id || 'guest'}`;
    const [forecast, setForecast] = useState<ForecastResponse | null>(null);
    const [showBounds, setShowBounds] = useState(false);

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

    const showForecast = (next: ForecastResponse) => {
        setForecast(next);
        setShowBounds(false);
        setIsLoading(false);
    };

    const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;
        setIsLoading(true);
        const started = Date.now();
        const waitForForecast = async () => {
            while (Date.now() - started < 30000) {
                await new Promise((resolve) => window.setTimeout(resolve, 2000));
                try {
                    const saved = await getForecast();
                    if ((saved.days?.length || 0) >= 30) return saved;
                } catch {
                    // сервер ещё считает
                }
            }
            throw new Error('Прогноз ещё не готов. Обновите страницу чуть позже.');
        };
        try {
            const uploaded = uploadStatement(file).catch(() => null);
            const ready = await Promise.race([
                uploaded.then((saved) => ((saved?.days?.length || 0) >= 30 ? saved : null)),
                waitForForecast(),
            ]);
            if (!ready) throw new Error('Прогноз ещё не готов. Обновите страницу чуть позже.');
            showForecast(ready);
            toast.success('Прогноз получен');
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Ошибка загрузки выписки');
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
            showForecast(await uploadStatement(file));
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

    return (
        <div className="DashboardContainer">

            {/* ШАПКА (Приветствие слева, кнопка/статус справа) */}
            <div className="DashboardHeader">
                <div>
                    <h1>Финансовый пульс</h1>
                    <p>Здравствуйте, {user?.company_name || user?.name || 'Гость'}. Ваш прогноз готов.</p>
                </div>
                {!!forecast?.days?.length && (
                    <button className="Button" onClick={handleReset}>
                        Загрузить другой файл
                    </button>
                )}
            </div>

            {!user && !loading && (
                <div className="GuestPulse" aria-hidden="true">
                    <div className="skeleton skeleton-title" />
                    <div className="skeleton skeleton-chart" />
                    <div className="skeleton skeleton-text" />
                    <div className="skeleton skeleton-text" />
                    <button className="Button" onClick={openModal}>Войти, чтобы увидеть пульс</button>
                </div>
            )}

            {/* ЗОНА ЗАГРУЗКИ */}
            {user && !forecast?.days?.length && !isLoading && (
                <div className="UploadZone">
                    <div className="UploadIcon">📊</div>
                    <h2>Загрузите выписку для анализа</h2>
                    <p>Поддерживаются форматы CSV (выгрузки из банка или 1С)</p>

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
                    <h2>ИИ анализирует транзакции...</h2>
                    <p>Пожалуйста, подождите</p>
                </div>
            )}

            {/* РЕЗУЛЬТАТЫ (Сетка) */}
            {forecast?.days?.length && !isLoading ? (
                <div className="ResultsGrid">

                    {/* Карточка бизнеса (Правая колонка, верх) */}
                    <div className="GridCard">
                        <PlasticCard
                            companyName={user?.company_name || user?.name || 'Моя компания'}
                            ostatok={rubles(forecast.days[forecast.days.length - 1]?.expected_balance || 0)}
                            cardStatus={
                                nearestGap(forecast.alerts)
                                    ? `Разрыв ${chancePercent(nearestGap(forecast.alerts)?.chance || 0)}%`
                                    : 'Стабилен'
                            }
                        />
                    </div>

                    {/* Регулярные платежи, которые нашёл бэкенд */}
                    <div className="GridTable">
                        <h3 className="SectionTitle">Регулярные платежи</h3>
                        {recurringPayments(forecast).length ? (
                            <div className="GapList">
                                {recurringByCategory(recurringPayments(forecast)).map((group) => (
                                    <div key={`${group.category}:${group.is_income}`} className="GapCard">
                                        <div>
                                            <div className="GapCard-date">{group.category}</div>
                                            <div className="GapCard-note">
                                                {group.count} {group.is_income ? 'приходов' : 'расходов'}
                                            </div>
                                        </div>
                                        <div className="GapCard-sum">
                                            <div className={group.is_income ? 'Amount--plus' : 'Amount--minus'}>
                                                {group.is_income ? '+' : '−'}{rubles(group.amount).toLocaleString('ru-RU')} ₽
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        ) : (
                            <p className="GapCard-note">Регулярных платежей в выписке не найдено.</p>
                        )}
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
                            showBounds={showBounds}
                            history={(forecast.history?.length
                                ? forecast.history
                                : [{ date: forecast.days[0].date, balance: forecast.days[0].expected_balance }]
                            ).map((point) => ({
                                date: point.date,
                                balance: rubles(point.balance),
                            }))}
                            forecast={forecast.days.map((day) => ({
                                date: day.date,
                                balance: rubles(day.expected_balance),
                                low: rubles(day.balance_low),
                                high: rubles(day.balance_high),
                            }))}
                        />
                        {showBounds && !!forecast.rare_payments?.length && (
                            <div className="GapList" style={{ marginTop: '12px' }}>
                                {forecast.rare_payments.map((payment) => (
                                    <div key={`${payment.is_income}:${payment.amount}`} className="GapCard">
                                        <div>
                                            <div className="GapCard-date">
                                                Редкий {payment.is_income ? 'приход' : 'платёж'} не входит в ежедневный расход
                                            </div>
                                            <div className="GapCard-note">
                                                около {rubles(payment.amount).toLocaleString('ru-RU')} ₽, замечен {payment.seen} раза за последние три месяца
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}
                        {!showBounds && (
                            <button className="Button" onClick={() => setShowBounds(true)}>
                                Обработать
                            </button>
                        )}
                    </div>

                    <div className="GridOps">
                        <h3 className="SectionTitle">Последние операции</h3>
                        {(forecast.operations || []).length ? (
                            <div className="GapList">
                                {(forecast.operations || []).slice(0, 10).map((op: StatementOperation, index) => (
                                    <div key={`${op.date}:${op.description}:${index}`} className="GapCard">
                                        <div>
                                            <div className="GapCard-date">{op.date}</div>
                                            <div className="GapCard-note">{op.category || op.description || 'Без категории'}</div>
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