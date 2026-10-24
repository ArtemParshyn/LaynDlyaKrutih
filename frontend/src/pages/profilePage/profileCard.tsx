import { useAuth } from '../../context/AuthContext';
import '../../App.css';
import LogoutButton from './LogoutButton';

const ProfileCard = () => {
  const { user } = useAuth();

  const formattedCreatedDate = user?.created_at || user?.createdAt
    ? new Date(user.created_at || user.createdAt || '').toLocaleDateString('ru-RU', {
        day: 'numeric',
        month: 'long',
        year: 'numeric',
      })
    : '—';

  const companyId = user?.id != null ? String(user.id).padStart(4, '0') : '—';

  return (
    <article className="ProfileCard">
      <header className="ProfileCard-head">
        <span className="ProfileCard-kicker">аккаунт</span>
        <LogoutButton compact />
      </header>

      <h1 className="ProfileCard-name">{user?.company_name || user?.name || 'Пользователь'}</h1>

      <dl className="ProfileCard-rows">
        <div>
          <dt>почта</dt>
          <dd>{user?.email || '—'}</dd>
        </div>
        <div>
          <dt>номер</dt>
          <dd>#{companyId}</dd>
        </div>
        <div>
          <dt>с нами</dt>
          <dd>{formattedCreatedDate}</dd>
        </div>
      </dl>
    </article>
  );
};

export default ProfileCard;
