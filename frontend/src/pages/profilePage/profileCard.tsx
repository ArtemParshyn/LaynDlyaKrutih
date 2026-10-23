import { useAuth } from '../../context/AuthContext';
import '../../App.css';
import LogoutButton from "./LogoutButton";

const ProfileCard = () => {
  const { user } = useAuth();

  const formattedCreatedDate = user?.created_at || user?.createdAt
    ? new Date(user.created_at || user.createdAt || '').toLocaleDateString('ru-RU', {
        day: 'numeric',
        month: 'long',
        year: 'numeric',
      })
    : '—';

  const avatarSrc = user?.imageUrl || '/images/profile.png';

  return (
    <div className="ProfileCard">
      <img
        src={avatarSrc}
        className="ProfilePage-avatar"
        alt={user?.company_name || 'Аватар пользователя'}
      />

      <div className="ProfileCard-info">
        <h1>{user?.company_name || user?.name || 'Пользователь'}</h1>
        <p><strong>Email:</strong> {user?.email}</p>
        <p><strong>ID компании:</strong> #{user?.id}</p>
        <p><strong>Дата регистрации:</strong> {formattedCreatedDate}</p>
      </div>

      <LogoutButton compact />
    </div>
  );
};

export default ProfileCard;