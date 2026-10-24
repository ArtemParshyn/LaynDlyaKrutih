import { useAuth } from '../context/AuthContext';
import { Link } from 'react-router-dom';
import '../App.css';

const SettingsPage = () => {
  const { user } = useAuth();

  return (
    <div className="SettingsPage">
      <h1 className="PageTitle">Настройки</h1>

      <div className="SettingsCard">
        <h3>Аккаунт</h3>
        <dl className="AccountCard-rows">
          <div>
            <dt>Компания</dt>
            <dd>{user?.company_name || user?.name || '—'}</dd>
          </div>
          <div>
            <dt>Почта</dt>
            <dd>{user?.email || '—'}</dd>
          </div>
        </dl>
        <Link to="/profile" className="Button">
          Открыть профиль
        </Link>
      </div>
    </div>
  );
};

export default SettingsPage;
