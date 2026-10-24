import { useAuth } from '../context/AuthContext';
import { Link } from 'react-router-dom';
import '../App.css';

const SettingsPage = () => {
  const { user } = useAuth();

  return (
    <div className="SettingsPage">
      <h1 className="PageTitle">Настройки и система</h1>

      <div className="SettingsCard">
        <h3>Текущий аккаунт</h3>
        <p><strong>Компания:</strong> {user?.company_name || user?.name || '—'}</p>
        <p><strong>Email:</strong> {user?.email || '—'}</p>
        <p><strong>ID в системе:</strong> #{user?.id ?? '—'}</p>
        <Link to="/profile" className="Button">
          Управление профилем
        </Link>
      </div>
    </div>
  );
};

export default SettingsPage;
