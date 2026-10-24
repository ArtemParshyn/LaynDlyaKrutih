import { useEffect, useState } from 'react';
import { useAuth } from '../../context/AuthContext';
import { ProfileCardSkeleton } from '../../components/skeleton';
import '../../App.css';
import ProfileCard from './profileCard';
import { ChangePasswordForm } from './ChangePasswordForm';
import EditNameForm from './EditNameForm';
import EmptyState from '../../components/EmptyState';
import { toast } from 'sonner';

const ProfilePage = () => {
  const { user, loading, openModal, deleteAccount, refreshProfile } = useAuth();
  const [isPasswordOpen, setIsPasswordOpen] = useState(false);
  const [isNameOpen, setIsNameOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

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
            <button
              className="Button ProfileActions-btn ProfileActions-btn--danger"
              onClick={handleDeleteAccount}
              disabled={isDeleting}
            >
              {isDeleting ? 'Удаление...' : 'Удалить аккаунт'}
            </button>
          </div>

          <EditNameForm isOpen={isNameOpen} onClose={() => setIsNameOpen(false)} />
          <ChangePasswordForm isOpen={isPasswordOpen} onClose={() => setIsPasswordOpen(false)} />
        </div>
      )}
    </div>
  );
};

export default ProfilePage;
