import { useAuth } from '../../context/AuthContext';
import '../../App.css';
import LogoutButton from './LogoutButton';

const ProfileCard = () => {
  const { user } = useAuth();

  const created = user?.created_at || user?.createdAt;
  const formattedCreatedDate = created
    ? new Date(created).toLocaleDateString('ru-RU', {
        day: 'numeric',
        month: 'long',
        year: 'numeric',
      })
    : '—';

  const companyName = user?.company_name || user?.name || 'Компания';

  return (
    <article className="AccountCard">
      <header className="AccountCard-head">
        <div>
          <span className="AccountCard-kicker">Компания</span>
          <h1 className="AccountCard-name">{companyName}</h1>
        </div>
        <LogoutButton compact />
      </header>

      <dl className="AccountCard-rows">
        <div>
          <dt>Почта</dt>
          <dd>{user?.email || '—'}</dd>
        </div>
        <div>
          <dt>В системе с</dt>
          <dd>{formattedCreatedDate}</dd>
        </div>
      </dl>
    </article>
  );
};

export default ProfileCard;
