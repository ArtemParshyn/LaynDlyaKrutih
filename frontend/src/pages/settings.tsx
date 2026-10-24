import { useAuth } from '../context/AuthContext';
import { Link } from 'react-router-dom';
import '../App.css';

const SettingsPage = () => {
  const { user } = useAuth();

  return (
    <div style={{ maxWidth: '800px', margin: '0 auto', padding: '20px' }}>
      <h1 className="PageTitle">Настройки и система</h1>

      <div style={{
        background: 'var(--surface)',
        borderRadius: '16px',
        padding: '24px',
        marginBottom: '24px',
        border: '1px solid var(--border)'
      }}>
        <h3 style={{ marginTop: 0 }}>Текущий аккаунт</h3>
        <p><strong>Компания:</strong> {user?.company_name || user?.name}</p>
        <p><strong>Email:</strong> {user?.email}</p>
        <p><strong>ID в системе:</strong> #{user?.id}</p>
        <Link to="/profile" className="Button" style={{ display: 'inline-block', marginTop: '10px' }}>
          Управление профилем
        </Link>
      </div>
    </div>
  );
};

export default SettingsPage;
