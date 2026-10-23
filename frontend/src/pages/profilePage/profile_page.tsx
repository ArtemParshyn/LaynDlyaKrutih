import { useState } from 'react';
import { useAuth } from '../../context/AuthContext';
import { ProfileCardSkeleton } from '../../components/skeleton';
import '../../App.css';
import ProfileCard from './profileCard';
import { ChangePasswordForm } from './ChangePasswordForm';
import EditAvatar from './EditAvatar';
import EditNameForm from './EditNameForm';
import EmptyState from '../../components/EmptyState';
import { toast } from 'sonner';

const ProfilePage = () => {
  const { user, loading, openModal, deleteAccount } = useAuth();
  const [isPasswordOpen, setIsPasswordOpen] = useState(false);
  const [openAvatar, setOpenAvatar] = useState(false);
  const [isNameOpen, setIsNameOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const handleOpenAvatarImage = () => {
    setOpenAvatar((prev) => !prev);
  };

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
        <>
          <ProfileCard />

          <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginTop: '20px' }}>
            <div>
              <button className="Button" onClick={handleOpenAvatarImage}>
                {openAvatar ? 'Скрыть загрузку аватарки' : 'Сменить аватарку'}
              </button>
              {openAvatar && <EditAvatar />}
            </div>

            <div>
              <button className="Button" onClick={() => setIsNameOpen(true)}>
                Сменить название компании
              </button>
              <EditNameForm isOpen={isNameOpen} onClose={() => setIsNameOpen(false)} />
            </div>

            <div>
              <button className="Button" onClick={() => setIsPasswordOpen(true)}>
                Сменить пароль
              </button>
              <ChangePasswordForm
                isOpen={isPasswordOpen}
                onClose={() => setIsPasswordOpen(false)}
              />
            </div>

            <div style={{ marginTop: '20px' }}>
              <button
                className="Button"
                onClick={handleDeleteAccount}
                disabled={isDeleting}
                style={{
                  backgroundColor: 'rgba(239, 68, 68, 0.15)',
                  color: '#ef4444',
                  borderColor: '#ef4444',
                }}
              >
                {isDeleting ? 'Удаление...' : 'Удалить аккаунт'}
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
};

export default ProfilePage;