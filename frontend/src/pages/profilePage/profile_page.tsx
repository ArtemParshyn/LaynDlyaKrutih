import { useEffect, useState } from 'react';
import { useAuth } from '../../context/AuthContext';
import { ProfileCardSkeleton } from '../../components/skeleton';
import '../../App.css';
import ProfileCard from './profileCard';
import { ChangePasswordForm } from './ChangePasswordForm';
import EditNameForm from './EditNameForm';
import EmptyState from '../../components/EmptyState';
import { toast } from 'sonner';
import { getOneCSettings, regenerateOneCSettings, OneCSettings } from '../../api/auth';

const ProfilePage = () => {
  const { user, loading, openModal, deleteAccount, refreshProfile } = useAuth();
  const [isPasswordOpen, setIsPasswordOpen] = useState(false);
  const [isNameOpen, setIsNameOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [oneC, setOneC] = useState<OneCSettings | null>(null);
  const [oneCOpen, setOneCOpen] = useState(false);
  const [oneCLoading, setOneCLoading] = useState(false);

  useEffect(() => {
    refreshProfile().catch(() => {});
  }, [refreshProfile]);

  const handleDeleteAccount = async () => {
    const confirmed = window.confirm(
      'Вы уверены, что хотите удалить свой аккаунт? Это действие необратимо и удалит данные из базы.'
    );
    if (!confirmed) return;

    setIsDeleting(true);
    try {
      await deleteAccount();
      toast.success('Аккаунт успешно удален');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Ошибка при удалении аккаунта');
    } finally {
      setIsDeleting(false);
    }
  };

  const toggleOneC = async () => {
    if (oneCOpen) { setOneCOpen(false); return; }
    setOneCLoading(true);
    try { setOneC(await getOneCSettings()); setOneCOpen(true); }
    catch (err) { toast.error(err instanceof Error ? err.message : 'Не удалось получить настройки 1С'); }
    finally { setOneCLoading(false); }
  };
  const regenerateOneC = async () => {
    if (!window.confirm('Старый токен 1С перестанет работать. Продолжить?')) return;
    try { setOneC(await regenerateOneCSettings()); toast.success('Токен 1С обновлён'); }
    catch (err) { toast.error(err instanceof Error ? err.message : 'Не удалось обновить токен'); }
  };
  const copy = async (value: string) => { await navigator.clipboard.writeText(value); toast.success('Ссылка скопирована'); };

  return (
    <div className="ProfilePage">
      {loading ? (
        <ProfileCardSkeleton />
      ) : !user ? (
        <EmptyState
          icon="🔒"
          title="Вы не авторизованы"
          description="Войдите в аккаунт, чтобы увидеть свой профиль."
          action={
            <button className="Button" onClick={openModal}>
              Войти
            </button>
          }
        />
      ) : (
        <div className="ProfileStage">
          <ProfileCard />

          <div className="ProfileActions">
            <button className="Button ProfileActions-btn" onClick={() => setIsNameOpen(true)}>
              Сменить название компании
            </button>
            <button className="Button ProfileActions-btn" onClick={() => setIsPasswordOpen(true)}>
              Сменить пароль
            </button>
            <button className="Button ProfileActions-btn" onClick={toggleOneC} disabled={oneCLoading}>
              {oneCLoading ? 'Загрузка...' : oneCOpen ? 'Скрыть API 1С' : 'API-кабинет 1С'}
            </button>
            <button
              className="Button ProfileActions-btn ProfileActions-btn--danger"
              onClick={handleDeleteAccount}
              disabled={isDeleting}
            >
              {isDeleting ? 'Удаление...' : 'Удалить аккаунт'}
            </button>
          </div>

          {oneCOpen && oneC && <section className="OneCApiCard">
            <h2>Подключение 1С</h2>
            <p>Выберите пульс и вставьте ссылку во внешнюю обработку 1С.</p>
            {oneC.pulses.map((pulse) => <div className="OneCApiRow" key={pulse.id}>
              <div><strong>{pulse.name}</strong><input readOnly value={pulse.upload_url} /></div>
              <button className="Button" onClick={() => copy(pulse.upload_url)}>Копировать</button>
            </div>)}
            {!oneC.pulses.length && <p>Сначала создайте пульс.</p>}
            <button className="Button" onClick={regenerateOneC}>Перегенерировать токен</button>
          </section>}

          <EditNameForm isOpen={isNameOpen} onClose={() => setIsNameOpen(false)} />
          <ChangePasswordForm isOpen={isPasswordOpen} onClose={() => setIsPasswordOpen(false)} />
        </div>
      )}
    </div>
  );
};

export default ProfilePage;
