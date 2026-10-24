import { useEffect, useState } from 'react';
import { useAuth } from '../context/AuthContext';
import '../App.css';
import FinanceChart from '../components/FinanceChart';
import { toast } from 'sonner';
import { Transaction } from '../utils/csv';
import { statementToTransactions, toBackendStatement } from '../utils/statement';
import { getForecast, uploadStatementText, ForecastResponse, rubles } from '../api/forecast';
import AnalyzedTable from "../components/AnalyzedTable";
import PlasticCard from '../components/BusinessCard/BuisnessCard';

const Dashboard = () => {
    const { user, loading, openModal } = useAuth();
    const [isLoading, setIsLoading] = useState(false);
    const [uploadTransactions, setUploadTransactions] = useState<Transaction[]>([]);

    // Храним данные графика и найденные разрывы
    const [forecast, setForecast] = useState<ForecastResponse | null>(null);

    const showForecast = (next: ForecastResponse, transactions?: Transaction[]) => {
        setForecast(next);
        if (transactions) setUploadTransactions(transactions);
        setIsLoading(false);
    };

    useEffect(() => {
        if (!user) return;
        getForecast()
            .then((saved) => {
                if (saved.days?.length) setForecast(saved);
            })
            .catch(() => {});
    }, [user]);

    const processData = async (text: string) => {
        try {
            const statement = toBackendStatement(text);
            const transactions = statementToTransactions(statement);
            if (transactions.length === 0) throw new Error('Файл пуст или неверный формат');
            showForecast(await uploadStatementText(statement), transactions);
            toast.success('Выписка отправлена, прогноз получен');
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Ошибка загрузки выписки');
            setIsLoading(false);
        }
    };

    // Обработка загруженного файла
    const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        setIsLoading(true);
        const reader = new FileReader();
        reader.onload = (event) => processData(String(event.target?.result || ''));
        reader.onerror = () => {
            toast.error('Не удалось прочитать файл');
            setIsLoading(false);
        };
        reader.readAsText(file);
    };

    // Загрузка демо-данных из public/data/financial_data.csv
    const loadDemoData = async () => {
        setIsLoading(true);
        try {
            const response = await fetch('/data/financial_data.csv');
            if (!response.ok) throw new Error('Демо-файл не найден');
            const text = await response.text();
            await processData(text);
        } catch (err) {
            toast.error('Не удалось загрузить демо-файл. Проверьте папку public/data');
            setIsLoading(false);
        }
    };

    // Сброс к начальному экрану
    const handleReset = () => {
        setForecast(null);
        setUploadTransactions([]);
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
                            cardStatus={forecast.alerts?.length ? 'Риск разрыва' : 'Стабилен'}
                        />
                    </div>

                    {/* Таблица (Левая колонка, под приветствием) */}
                    <div className="GridTable">
                        <h3 className="SectionTitle">Детализация операций</h3>
                        <AnalyzedTable transactions={uploadTransactions} />
                    </div>

                    {/* График (На всю ширину, или по центру) */}
                    <div className="GridChart">
                        {/* Умный баннер (Перенесен над графиком для логики) */}
                        {forecast.rough && <p>Прогноз ориентировочный: истории меньше 30 дней.</p>}
                        {!!forecast.alerts?.length ? (
                            <div className="SuccessBanner" style={{
                                borderColor: 'var(--error)',
                                color: 'var(--error)',
                                background: 'rgba(244, 63, 94, 0.1)',
                                marginBottom: '20px',
                                width: '100%'
                            }}>
                                ⚠️ Внимание! Обнаружено {forecast.alerts.length} риск(ов) кассового разрыва.
                            </div>
                        ) : (
                            <div className="SuccessBanner" style={{ marginBottom: '20px', width: '100%' }}>
                                ✅ Кассовых разрывов не обнаружено. Прогноз стабилен.
                            </div>
                        )}

                        <FinanceChart data={forecast.days.map((day) => ({
                            date: day.date,
                            balance: rubles(day.expected_balance),
                            low: rubles(day.balance_low),
                            high: rubles(day.balance_high),
                        }))} />
                    </div>

                    {/* Предупреждения (Снизу на всю ширину) */}
                    {!!forecast.alerts?.length && (
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
                                                {alert.reason}
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