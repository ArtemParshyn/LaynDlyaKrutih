import { FormEvent, useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { toast } from 'sonner';
import { useAuth } from '../context/AuthContext';
import { rubles } from '../api/forecast';
import { PulseCard, createPulse, deletePulse, listPulses } from '../api/pulses';
import '../App.css';

const money = (kopecks: number) =>
    `${rubles(kopecks).toLocaleString('ru-RU', { maximumFractionDigits: 0 })} ₽`;

const day = (value?: string) => {
    if (!value) return 'Нет выписки';
    const [year, month, date] = value.split('-');
    return year && month && date ? `${date}.${month}.${year}` : value;
};

const Sparkline = ({ points }: { points: number[] }) => {
    if (points.length < 2) {
        return <div className="PulseSpark PulseSpark--empty">Линия появится после выписки</div>;
    }
    const min = Math.min(...points);
    const max = Math.max(...points);
    const span = max - min || 1;
    const width = 280;
    const height = 56;
    const path = points.map((value, index) => {
        const x = (index / (points.length - 1)) * width;
        const y = height - 4 - ((value - min) / span) * (height - 8);
        return `${index === 0 ? 'M' : 'L'}${x.toFixed(1)} ${y.toFixed(1)}`;
    }).join(' ');
    return (
        <svg className="PulseSpark" viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" aria-hidden="true">
            <path d={path} fill="none" stroke="currentColor" strokeWidth="2" />
        </svg>
    );
};

const Pulses = () => {
    const { user, loading, openModal } = useAuth();
    const navigate = useNavigate();
    const [pulses, setPulses] = useState<PulseCard[]>([]);
    const [ready, setReady] = useState(false);
    const [name, setName] = useState('');
    const [file, setFile] = useState<File | null>(null);
    const [saving, setSaving] = useState(false);

    const load = async () => {
        const next = await listPulses();
        setPulses(next || []);
        setReady(true);
    };

    useEffect(() => {
        if (!user) {
            setReady(true);
            return;
        }
        load().catch((err) => {
            toast.error(err instanceof Error ? err.message : 'Не удалось открыть кабинет');
            setReady(true);
        });
    }, [user]);

    const handleCreate = async (event: FormEvent) => {
        event.preventDefault();
        const title = name.trim();
        if (!title || !file) {
            toast.error('Укажите имя и файл CSV');
            return;
        }
        setSaving(true);
        try {
            const created = await createPulse(title, file);
            setName('');
            setFile(null);
            navigate(`/pulses/${created.id}`, { state: { fresh: true } });
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Не удалось создать пульс');
            setSaving(false);
        }
    };

    const handleDelete = async (pulse: PulseCard) => {
        if (!window.confirm(`Удалить «${pulse.name}» вместе с выпиской и прогнозом?`)) return;
        try {
            await deletePulse(pulse.id);
            setPulses((current) => current.filter((item) => item.id !== pulse.id));
            toast.success('Пульс удалён');
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Не удалось удалить пульс');
        }
    };

    if (!user && !loading) {
        return (
            <div className="PulseBoard">
                <div className="DashboardHeader">
                    <div>
                        <h1>Пульсы</h1>
                        <p>Отдельный график для каждой выписки.</p>
                    </div>
                </div>
                <button className="Button" type="button" onClick={openModal}>Войти</button>
            </div>
        );
    }

    return (
        <div className="PulseBoard">
            <div className="DashboardHeader">
                <div>
                    <h1>Пульсы</h1>
                    <p>{user?.company_name || user?.name || 'Кабинет'}. Одна карточка — один график.</p>
                </div>
            </div>

            <form className="PulseCreate" onSubmit={handleCreate}>
                <label>
                    Имя
                    <input
                        type="text"
                        maxLength={80}
                        value={name}
                        onChange={(event) => setName(event.target.value)}
                        placeholder="Магазин на Ленина"
                        required
                    />
                </label>
                <label className="PulseCreate-file">
                    {file ? file.name : 'Выбрать CSV'}
                    <input
                        type="file"
                        accept=".csv"
                        onChange={(event) => setFile(event.target.files?.[0] || null)}
                        required
                    />
                </label>
                <button className="Button upload-btn" type="submit" disabled={saving}>
                    {saving ? 'Создаём' : 'Новый пульс'}
                </button>
            </form>

            {!ready ? <p className="GapCard-note">Открываем кабинет.</p> : null}
            {ready && !pulses.length ? (
                <p className="GapCard-note">Пока пусто. Первая выписка станет первой карточкой.</p>
            ) : null}

            <div className="PulseGrid">
                {pulses.map((pulse) => (
                    <article key={pulse.id} className="PulseCard">
                        <div className="PulseCard-top">
                            <h2>{pulse.name}</h2>
                            <button type="button" className="PulseCard-delete" onClick={() => handleDelete(pulse)}>
                                Удалить
                            </button>
                        </div>
                        <Sparkline points={(pulse.history || []).map((point) => point.balance)} />
                        <div className="PulseCard-balance">{pulse.as_of ? money(pulse.balance) : '—'}</div>
                        <div className="PulseCard-date">на {day(pulse.as_of)}</div>
                        <Link className="Button" to={`/pulses/${pulse.id}`}>Открыть</Link>
                    </article>
                ))}
            </div>
        </div>
    );
};

export default Pulses;
