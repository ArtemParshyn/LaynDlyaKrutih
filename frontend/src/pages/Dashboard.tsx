import { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import '../App.css';
import FinanceChart from '../components/FinanceChart';
import { toast } from 'sonner';
import { parseCSV } from '../utils/csv';
import { forecastBalance, CashGap, ForecastPoint } from '../utils/forecast';
import AnalyzedTable from "../components/AnalyzedTable";
import PlasticCard from '../components/BusinessCard/BuisnessCard';

const Dashboard = () => {
    const { user } = useAuth();
    const [isLoading, setIsLoading] = useState(false);
    const [uploadTransactions, setUploadTransactions] = useState<any[]>([]);

    // Храним данные графика и найденные разрывы
    const [chartData, setChartData] = useState<ForecastPoint[] | null>(null);
    const [gaps, setGaps] = useState<CashGap[]>([]);

    // Общая функция для обработки текста (вызывается и для файла, и для демо)
    const processData = (text: string) => {
        try {
            const transactions = parseCSV(text);

            if (transactions.length === 0) {
                throw new Error('Файл пуст или неверный формат');
            }

            // Считаем прогноз. Стартовый баланс возьмем 50000
            const { forecast, gaps: detectedGaps } = forecastBalance(transactions, 50000);

            // Небольшая задержка для красоты (имитация работы ИИ)
            setTimeout(() => {
                setChartData(forecast);
                setGaps(detectedGaps);
                setUploadTransactions(transactions);
                setIsLoading(false);
                toast.success('Анализ завершен!');
            }, 1500);

        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Ошибка парсинга файла');
            setIsLoading(false);
            setUploadTransactions([]);
        }
    };

    // Обработка загруженного файла
    const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        setIsLoading(true);
        const reader = new FileReader();

        reader.onload = (event) => {
            const text = event.target?.result as string;
            processData(text);
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
            processData(text);
        } catch (err) {
            toast.error('Не удалось загрузить демо-файл. Проверьте папку public/data');
            setIsLoading(false);
        }
    };

    // Сброс к начальному экрану
    const handleReset = () => {
        setChartData(null);
        setGaps([]);
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
                {chartData && (
                    <button className="Button" onClick={handleReset}>
                        Загрузить другой файл
                    </button>
                )}
            </div>

            {/* ЗОНА ЗАГРУЗКИ */}
            {!chartData && !isLoading && (
                <div className="UploadZone">
                    <div className="UploadIcon">📊</div>
                    <h2>Загрузите выписку для анализа</h2>
                    <p>Поддерживаются форматы CSV (выгрузки из банка или 1С)</p>

                    <div style={{
                        display: 'flex',
                        gap: '15px',
                        marginTop: '20px',
                        flexWrap: 'wrap',
                        justifyContent: 'center'
                    }}>
                        <label className="Button upload-btn" style={{ margin: 0 }}>
                            Выбрать файл .CSV
                            <input type="file" accept=".csv" onChange={handleFileUpload} hidden />
                        </label>

                        <button
                            className="Button"
                            onClick={loadDemoData}
                            style={{
                                background: 'var(--surface-button)',
                                color: 'var(--text)',
                                border: '1px solid var(--border)'
                            }}
                        >
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
            {chartData && !isLoading && (
                <div className="ResultsGrid">

                    {/* Карточка бизнеса (Правая колонка, верх) */}
                    <div className="GridCard">
                        <PlasticCard
                            companyName={user?.company_name || user?.name || 'Моя компания'}
                            ostatok={chartData[chartData.length - 1]?.balance || 0}
                            cardStatus={gaps.length > 0 ? 'Риск разрыва' : 'Стабилен'}
                        />
                    </div>

                    {/* Таблица (Левая колонка, под приветствием) */}
                    <div className="GridTable">
                        <h3 style={{ color: 'var(--text)', marginTop: 0, marginBottom: '15px' }}>Детализация операций</h3>
                        <AnalyzedTable transactions={uploadTransactions} />
                    </div>

                    {/* График (На всю ширину, или по центру) */}
                    <div className="GridChart">
                        {/* Умный баннер (Перенесен над графиком для логики) */}
                        {gaps.length > 0 ? (
                            <div className="SuccessBanner" style={{
                                borderColor: 'var(--error)',
                                color: 'var(--error)',
                                background: 'rgba(244, 63, 94, 0.1)',
                                marginBottom: '20px',
                                width: '100%'
                            }}>
                                ⚠️ Внимание! Обнаружено {gaps.length} риск(ов) кассового разрыва.
                            </div>
                        ) : (
                            <div className="SuccessBanner" style={{ marginBottom: '20px', width: '100%' }}>
                                ✅ Кассовых разрывов не обнаружено. Прогноз стабилен.
                            </div>
                        )}

                        <FinanceChart data={chartData} />
                    </div>

                    {/* Предупреждения (Снизу на всю ширину) */}
                    {gaps.length > 0 && (
                        <div className="GridWarnings">
                            <h3 style={{ color: 'var(--text)', marginBottom: '15px' }}>Предупреждения о разрывах:</h3>
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                                {gaps.map((gap, idx) => (
                                    <div key={idx} style={{
                                        background: 'var(--surface)',
                                        padding: '15px',
                                        borderRadius: '12px',
                                        borderLeft: '4px solid var(--error)',
                                        display: 'flex',
                                        justifyContent: 'space-between',
                                        alignItems: 'center',
                                        flexWrap: 'wrap',
                                        gap: '10px'
                                    }}>
                                        <div>
                                            <div style={{ fontWeight: 'bold', color: 'var(--text)', fontSize: '16px' }}>
                                                📅 {gap.date}
                                            </div>
                                            <div style={{
                                                color: 'var(--text-muted)',
                                                fontSize: '14px',
                                                marginTop: '4px'
                                            }}>
                                                Нехватка средств для обязательных платежей
                                            </div>
                                        </div>
                                        <div style={{ textAlign: 'right' }}>
                                            <div style={{ color: 'var(--error)', fontWeight: 'bold', fontSize: '18px' }}>
                                                -{gap.shortage.toLocaleString('ru-RU')} ₽
                                            </div>
                                            <div style={{
                                                color: 'var(--text-faint)',
                                                fontSize: '12px',
                                                marginTop: '4px'
                                            }}>
                                                Вероятность: {Math.round(gap.probability * 100)}%
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                </div>
            )}
        </div>
    );
};

export default Dashboard;